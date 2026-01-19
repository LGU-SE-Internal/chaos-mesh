// Copyright 2021 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package fusedev

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pingcap/errors"
)

// GrantAccess appends 'c 10:229 rwm' to devices.allow
func GrantAccess() error {
	pid := os.Getpid()
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)

	cgroupFile, err := os.Open(cgroupPath)
	if err != nil {
		return err
	}
	defer cgroupFile.Close()

	// TODO: encapsulate these logic with chaos-daemon StressChaos part
	cgroupScanner := bufio.NewScanner(cgroupFile)
	var deviceCgroupPath string
	var isCgroupV2 bool
	for cgroupScanner.Scan() {
		var (
			text  = cgroupScanner.Text()
			parts = strings.SplitN(text, ":", 3)
		)
		if len(parts) < 3 {
			return errors.Errorf("invalid cgroup entry: %q", text)
		}

		// cgroup v1: controllers are in parts[1], e.g. "devices"
		// cgroup v2: parts[1] is empty and unified hierarchy path is in parts[2]
		// Prefer v1 "devices" entry if present; only use v2 unified entry if no v1 found
		if strings.Contains(parts[1], "devices") {
			deviceCgroupPath = parts[2]
			isCgroupV2 = false
		} else if parts[1] == "" && len(deviceCgroupPath) == 0 {
			// unified cgroup v2 entry like: "0::/user.slice/..."
			// Only set if we haven't found a v1 devices entry yet
			deviceCgroupPath = parts[2]
			isCgroupV2 = true
		}
	}

	if err := cgroupScanner.Err(); err != nil {
		return err
	}

	if len(deviceCgroupPath) == 0 {
		return errors.New("fail to find device cgroup")
	}

	// It's hard to use /pkg/chaosdaemon/cgroups to wrap this logic.
	// For cgroup v1 the devices controller is usually mounted under
	// /sys/fs/cgroup/devices, while cgroup v2 uses a unified mount
	// under /sys/fs/cgroup. The host's fs is exposed under /host-sys.
	var finalPath string
	if isCgroupV2 {
		// For cgroup v2, check if devices controller is available/enabled
		// In cgroup v2, the devices controller may not be enabled on all systems
		// First try to read from /host-sys (in container context) or fallback to /sys (on host)
		var controllersPath string
		if _, err := os.Stat("/host-sys/fs/cgroup/cgroup.controllers"); err == nil {
			controllersPath = "/host-sys/fs/cgroup/cgroup.controllers"
		} else {
			controllersPath = "/sys/fs/cgroup/cgroup.controllers"
		}

		if controllers, err := os.ReadFile(controllersPath); err == nil && !strings.Contains(string(controllers), "devices") {
			// devices controller not enabled in cgroup v2, skip device granting
			// This is not an error - it's a valid configuration where device isolation
			// is not enforced through cgroup devices controller
			return nil
		}

		// Avoid double slashes when deviceCgroupPath is "/"
		if deviceCgroupPath == "/" {
			finalPath = "/host-sys/fs/cgroup/devices.allow"
		} else {
			finalPath = "/host-sys/fs/cgroup" + deviceCgroupPath + "/devices.allow"
		}
	} else {
		// Avoid double slashes when deviceCgroupPath is "/"
		if deviceCgroupPath == "/" {
			finalPath = "/host-sys/fs/cgroup/devices/devices.allow"
		} else {
			finalPath = "/host-sys/fs/cgroup/devices" + deviceCgroupPath + "/devices.allow"
		}
	}
	f, err := os.OpenFile(finalPath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	// 10, 229 according to https://www.kernel.org/doc/Documentation/admin-guide/devices.txt
	content := "c 10:229 rwm"
	_, err = f.WriteString(content)
	return err
}
