# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Chaos Mesh is a cloud-native Chaos Engineering platform for Kubernetes. It consists of:

- **Chaos Controller Manager**: Schedules and manages chaos experiments via CRD controllers (Workflow, Scheduler, and various fault type controllers)
- **Chaos Daemon**: Runs as a privileged DaemonSet that injects faults by interfacing with target Pod namespaces
- **Chaos Dashboard**: Web UI for managing and monitoring chaos experiments (React + TypeScript + Redux + Material UI)

## Development Commands

### Building and Testing

```bash
# Run all prerequisite checks (generate, vet, lint, fmt, tidy, install.sh, helm-values-schema)
make check

# Run unit tests
make test

# Run tests for a single package
go test ./pkg/path/to/package

# Build all container images (chaos-controller-manager, chaos-daemon, chaos-dashboard)
make image

# Generate all code (CRD manifests, deepcopy, client code, swagger spec)
make generate

# Generate CRD manifests only
make manifests/crd.yaml
```

### Code Generation

```bash
# Generate CRD manifests with controller-gen
make config

# Generate deepcopy files for CRD types
make generate-deepcopy

# Generate clientset, informer, and lister code
make generate-client

# Generate chaos-builder code for CustomResource Kinds
make chaos-build

# Generate protobuf files
make proto

# Generate swagger spec for dashboard API
make swagger_spec
```

### Linting and Formatting

```bash
# Format go files with goimports
make fmt

# Lint with revive
make lint

# Run go vet
make vet

# Run go mod tidy across all modules
make tidy
```

### UI Development

```bash
# Install UI dependencies
cd ui && pnpm install --frozen-lockfile

# Start UI development server (requires REACT_APP_API_URL)
cd ui && REACT_APP_API_URL=http://localhost:2333 BROWSER=none pnpm -F @ui/app start

# Build UI for production (requires UI=1 flag)
UI=1 make ui
```

### E2E Testing

```bash
# Build e2e test binaries
make e2e-build

# Run e2e tests (requires running Kubernetes cluster)
make e2e
```

### Development Environments

```bash
# Enter build environment shell
make enter-buildenv

# Enter development environment shell
make enter-devenv
```

## Architecture

### Controller Design Principles

Controllers in Chaos Mesh follow these principles (see controllers/README.md):

1. **One controller per field**: Each field should be controlled by at most one controller to avoid conflicts
2. **Standalone operation**: Controllers should work independently without dependencies on other controllers
3. **Error handling**: Return `ctrl.Result{Requeue: true}, nil` for retriable errors to leverage exponential backoff

### Directory Structure

- `api/v1alpha1/`: CRD type definitions and webhook implementations
- `cmd/`: Entry points for binaries (chaos-controller-manager, chaos-daemon, chaos-dashboard, etc.)
- `controllers/`: Kubernetes controllers organized by responsibility
  - `controllers/chaosimpl/`: Implementations for each chaos type (podchaos, networkchaos, iochaos, timechaos, stresschaos, etc.)
  - `controllers/common/`: Shared controller utilities
  - `controllers/schedule/`: Scheduler controller
  - `controllers/statuscheck/`: Status check controller
- `pkg/`: Shared libraries and utilities
  - `pkg/chaosdaemon/`: Chaos daemon gRPC service implementation
  - `pkg/dashboard/`: Dashboard API handlers
  - `pkg/client/`: Generated Kubernetes client code
- `helm/chaos-mesh/`: Helm chart for deployment
- `ui/`: React-based dashboard frontend (monorepo with app and packages)
- `e2e-test/`: End-to-end test suite

### Chaos Types

Chaos Mesh supports multiple fault injection types, each with its own controller and implementation:

- **PodChaos**: Pod-level faults (kill, failure, etc.)
- **NetworkChaos**: Network faults (delay, loss, corruption, partition)
- **IOChaos**: I/O faults (latency, errno injection)
- **TimeChaos**: Clock skew simulation
- **StressChaos**: CPU/memory stress
- **HTTPChaos**: HTTP fault injection
- **DNSChaos**: DNS fault injection
- **KernelChaos**: Kernel fault injection
- **JVMChaos**: JVM fault injection
- **AWSChaos, AzureChaos, GCPChaos**: Cloud provider fault injection
- **BlockChaos**: Block device faults
- **PhysicalMachineChaos**: Physical machine faults

### Code Generation Workflow

When modifying CRD types in `api/v1alpha1/`:

1. Update the Go struct definitions
2. Run `make generate` to regenerate all derived code
3. Run `make manifests/crd.yaml` to update CRD manifests
4. Run `make check` to verify formatting and linting

### Plural Exceptions

When generating client code, these resources use non-standard pluralization:
- PodChaos → podchaos
- HTTPChaos → httpchaos
- IOChaos → iochaos
- AWSChaos → awschaos
- JVMChaos → jvmchaos
- StressChaos → stresschaos
- AzureChaos → azurechaos
- PodHttpChaos → podhttpchaos
- GCPChaos → gcpchaos
- NetworkChaos → networkchaos
- KernelChaos → kernelchaos
- TimeChaos → timechaos
- BlockChaos → blockchaos
- PodIOChaos → podiochaos
- PodNetworkChaos → podnetworkchaos

## Development Workflow

1. Fork and clone the repository
2. Create a feature branch from master
3. Make changes and run `make check` to verify
4. Run `make test` to ensure unit tests pass
5. Test manually in a Kubernetes cluster (see [Configure Development Environment](https://chaos-mesh.org/docs/configure-development-environment/))
6. Commit with `--signoff` flag
7. Submit PR against master branch

### Adding a New Chaos Type

See [docs/adding-new-chaos-types.md](docs/adding-new-chaos-types.md) for a comprehensive guide on implementing new chaos injection types.

## Build System

The build system uses Docker-based build and dev environments:

- Build environment images are tagged per-branch (see `hack/env-image-tag.sh`)
- Most make targets run inside containerized environments via `RUN_IN_DEV_SHELL` or `RUN_IN_BUILD_SHELL`
- Local builds can be done with `local/` prefix targets (e.g., `make local/chaos-daemon`)
- Generated makefiles: `binary.generated.mk`, `container-image.generated.mk`, `local-binary.generated.mk`

## Testing

- Unit tests use failpoint injection (enable with `make failpoint-enable`, disable with `make failpoint-disable`)
- Test utilities are built with `make test-utils` (timer, multithread_tracee, fake clock objects)
- E2E tests require a running Kubernetes cluster and use Ginkgo framework
- Coverage reports generated with `make coverage`
