# RFC: BPFTime-based Userspace Fault Injection in Chaos Mesh

## Summary

This RFC proposes adding new fault injection capabilities to Chaos Mesh leveraging [bpftime](https://github.com/eunomia-bpf/bpftime), a userspace eBPF runtime. By utilizing bpftime's ability to hook and instrument userspace functions without kernel privileges, we can introduce a new chaos type that enables fault injection scenarios previously impossible or impractical with kernel-based approaches.

## Motivation

### Background

eBPF (extended Berkeley Packet Filter) has revolutionized observability and networking in Linux by allowing safe, efficient code execution in the kernel. However, traditional eBPF has limitations:

1. **Kernel Privileges Required**: Traditional eBPF requires kernel access and elevated privileges
2. **Limited to Kernel Space**: Cannot directly hook userspace functions efficiently
3. **Performance Overhead**: Switching between user and kernel space introduces latency
4. **Deployment Complexity**: Requires kernel module support and specific kernel versions

### What is bpftime?

[bpftime](https://github.com/eunomia-bpf/bpftime) is a userspace eBPF runtime that enables:

- **Userspace Function Hooking**: Attach eBPF programs to userspace functions (uprobe/uretprobe)
- **No Kernel Privileges**: Runs entirely in userspace without requiring root or kernel modules
- **High Performance**: Uses binary rewriting and JIT compilation for minimal overhead
- **Syscall Interception**: Can intercept and modify system calls
- **Dynamic Instrumentation**: Attach/detach hooks at runtime without restarting applications

#### bpftime Core Capabilities

Based on the [bpftime documentation](https://github.com/eunomia-bpf/bpftime) and [technical paper](https://arxiv.org/abs/2311.07923), bpftime provides the following capabilities that enable fault injection:

1. **Uprobe/Uretprobe Support** ([source](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md))
   - Attach to function entry and exit points
   - Read/modify function arguments and return values
   - Works with both dynamically linked libraries and static binaries
   - Supports symbol resolution via ELF symbol tables

2. **Syscall Tracing** ([source](https://github.com/eunomia-bpf/bpftime/blob/main/docs/syscall-tracing.md))
   - Intercept system calls in userspace
   - Modify syscall parameters before execution
   - Override syscall return values
   - Implemented via binary rewriting of syscall instructions

3. **eBPF Maps** ([source](https://ebpf.io/what-is-ebpf/#maps))
   - Share state between eBPF programs and userspace
   - Store configuration (probability, delay values, error codes)
   - Track injection statistics and metrics
   - Support hash maps, array maps, and ring buffers

4. **JIT Compilation** ([source](https://github.com/eunomia-bpf/bpftime/blob/main/vm/llvm-jit/README.md))
   - LLVM-based JIT for optimal performance
   - Compiles eBPF bytecode to native machine code
   - Reduces overhead to <5% for most operations
   - Supports AOT compilation for further optimization

5. **Shared Memory Communication** ([source](https://github.com/eunomia-bpf/bpftime/blob/main/docs/build-and-test.md))
   - Efficient IPC between injected agent and control plane
   - Ring buffer for event streaming
   - Perf event arrays for metrics collection

### Why Chaos Mesh Needs This

Current Chaos Mesh fault injection types (NetworkChaos, IOChaos, etc.) primarily operate at:
- Network layer (packet manipulation)
- Kernel layer (syscall failures, kernel faults)
- Container/Pod level (resource stress, time skew)

There's a gap in **application-layer fault injection** where we cannot:
1. Inject faults into specific library function calls (malloc, free, pthread functions)
2. Simulate failures in third-party library dependencies
3. Test error handling for specific code paths in compiled binaries
4. Inject latency into userspace function calls
5. Modify function return values or arguments dynamically

## Proposed Solution

### New Chaos Type: UserspaceChaos

Introduce a new CRD `UserspaceChaos` that leverages bpftime to inject faults into userspace applications.

### Use Cases and Fault Scenarios

#### 1. Memory Allocation Failures

**Scenario**: Test how applications handle `malloc()` or `new` operator failures.

**Use Case**: 
- Verify graceful degradation when memory is exhausted
- Test memory leak detection mechanisms
- Validate error handling in memory-intensive operations

**Implementation**: Hook `malloc`, `calloc`, `realloc` functions and return NULL with specified probability.

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: malloc-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-service
  mode: one
  duration: "30s"
  functionHook:
    function: "malloc"
    library: "libc.so.6"
    action: fail
    probability: 10  # 10% failure rate
    returnValue: "0"  # NULL pointer
```

#### 2. File I/O Function Failures

**Scenario**: Simulate failures in file operations at the libc level.

**Use Case**:
- Test error handling for `fopen`, `fread`, `fwrite` failures
- Simulate disk full conditions at application level
- Verify proper file descriptor cleanup

**Implementation**: Hook file I/O functions and return error codes.

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: file-io-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: database
  mode: all
  duration: "1m"
  functionHook:
    function: "fopen"
    library: "libc.so.6"
    action: fail
    filter:
      arguments:
        - index: 0  # filename parameter
          contains: "/data/"  # only affect /data/ paths
    returnValue: "0"  # NULL
    errno: "ENOSPC"  # No space left on device
```

#### 3. Network Library Function Delays

**Scenario**: Inject latency into specific network library calls.

**Use Case**:
- Test timeout handling in HTTP clients
- Verify retry logic in gRPC connections
- Simulate slow DNS resolution

**Implementation**: Hook functions like `connect()`, `send()`, `recv()` and add artificial delays.

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: network-latency
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: microservice-a
  mode: one
  duration: "2m"
  functionHook:
    function: "connect"
    library: "libc.so.6"
    action: delay
    delayMs: 5000  # 5 second delay
    probability: 50
```

#### 4. Threading and Synchronization Failures

**Scenario**: Test concurrent code by injecting failures in pthread operations.

**Use Case**:
- Verify deadlock detection mechanisms
- Test mutex/lock timeout handling
- Validate thread pool error recovery

**Implementation**: Hook `pthread_create`, `pthread_mutex_lock` and simulate failures.

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: pthread-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: worker-pool
  mode: one
  duration: "1m"
  functionHook:
    function: "pthread_create"
    library: "libpthread.so.0"
    action: fail
    probability: 5
    returnValue: "11"  # EAGAIN error
```

#### 5. Custom Application Function Hooking

**Scenario**: Hook specific functions in your application binary.

**Use Case**:
- Test specific error paths in your code
- Inject failures in internal API calls
- Simulate third-party SDK failures

**Implementation**: Hook named functions in the application binary using symbol names.

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: custom-function-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-app
  containerNames:
    - main-container
  mode: one
  duration: "30s"
  functionHook:
    function: "_Z15processRequestRKNSt7__cxx1112basic_stringE"  # mangled C++ function
    binary: "/app/myservice"
    action: modifyReturn
    returnValue: "-1"  # error code
    probability: 20
```

#### 6. SSL/TLS Library Failures

**Scenario**: Inject failures into OpenSSL/TLS library functions.

**Use Case**:
- Test certificate validation error handling
- Verify TLS connection retry logic
- Simulate SSL handshake failures

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: ssl-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: api-gateway
  mode: one
  duration: "1m"
  functionHook:
    function: "SSL_connect"
    library: "libssl.so.1.1"
    action: fail
    returnValue: "-1"
    probability: 10
```

#### 7. Database Client Library Failures

**Scenario**: Inject failures into database client libraries (MySQL, PostgreSQL, Redis clients).

**Use Case**:
- Test connection pool exhaustion handling
- Verify query timeout and retry mechanisms
- Simulate database unavailability

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: db-client-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: api-service
  mode: one
  duration: "2m"
  functionHook:
    function: "mysql_real_connect"
    library: "libmysqlclient.so"
    action: fail
    returnValue: "0"
    probability: 15
```

#### 8. Random Number Generation Manipulation

**Scenario**: Control randomness for deterministic testing.

**Use Case**:
- Test code that relies on randomness
- Reproduce specific random scenarios
- Validate shuffle/selection algorithms

**Example Configuration**:
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: fixed-random
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: simulation
  mode: one
  duration: "5m"
  functionHook:
    function: "rand"
    library: "libc.so.6"
    action: modifyReturn
    returnValue: "42"  # always return 42
    probability: 100
```

### How to Find and Specify Hook Points

One of the key challenges in userspace fault injection is identifying the correct hook points. This section details the methodologies and tools for discovering hookable functions.

#### 1. Symbol Discovery Methods

**For Dynamically Linked Libraries:**

```bash
# List all exported symbols in a library
nm -D /lib/x86_64-linux-gnu/libc.so.6 | grep " T "

# Example output:
# 00000000000a2d50 T malloc
# 00000000000a4820 T free
# 00000000000a2e30 T calloc

# Use readelf for detailed symbol information
readelf -s /lib/x86_64-linux-gnu/libc.so.6 | grep FUNC

# Use objdump for symbol addresses
objdump -T /lib/x86_64-linux-gnu/libc.so.6 | grep malloc
```

**For Static Binaries:**

```bash
# List all symbols in a binary (requires symbols not stripped)
nm /usr/bin/myapp

# For stripped binaries, use dynamic analysis
gdb /usr/bin/myapp
(gdb) info functions
```

**For C++ Applications (Mangled Names):**

```bash
# C++ function names are mangled - use c++filt to decode
nm /usr/bin/myapp | c++filt

# Example: _Z15processRequestRKNSt7__cxx1112basic_stringE
# Becomes: processRequest(std::__cxx11::basic_string const&)

# Find specific functions
nm /usr/bin/myapp | c++filt | grep "processRequest"
```

#### 2. Hook Point Specification Format

The `functionHook` configuration uses the following resolution order:

1. **Library Functions** (highest priority if `library` is specified):
   ```yaml
   functionHook:
     function: "malloc"
     library: "libc.so.6"  # Resolved via /etc/ld.so.cache and dlopen()
   ```
   - bpftime searches in standard library paths: `/lib`, `/usr/lib`, `/lib64`, `/usr/lib64`
   - Uses `dlopen()` to load the library and `dlsym()` to find the symbol
   - Supports library versioning (e.g., `libc.so.6` vs `libc.so`)

2. **Binary Functions** (if `binary` is specified):
   ```yaml
   functionHook:
     function: "myCustomFunction"
     binary: "/app/myservice"  # Absolute path within container
   ```
   - Resolves symbols from the specified binary's symbol table
   - Requires the binary to have debug symbols or non-stripped symbols
   - For position-independent executables (PIE), addresses are rebased at runtime

3. **Main Binary** (if neither `library` nor `binary` is specified):
   ```yaml
   functionHook:
     function: "main"  # Hooks main() of the target process
   ```
   - Attaches to the primary executable of the target process
   - Retrieved via `/proc/<pid>/exe`

#### 3. bpftime Hook Attachment Process

Based on [bpftime's uprobe implementation](https://github.com/eunomia-bpf/bpftime/blob/main/attach/frida_uprobe_attach/src/frida_uprobe_attach.cpp):

```
1. Symbol Resolution
   └─> Read ELF headers from target library/binary
   └─> Parse .dynsym and .symtab sections
   └─> Find symbol offset from base address

2. Address Calculation
   └─> Get library base address from /proc/<pid>/maps
   └─> Calculate absolute address: base_addr + symbol_offset

3. Hook Installation (using Frida)
   └─> Use Interceptor.attach() at calculated address
   └─> Install inline hook with trampoline
   └─> Original instruction bytes saved for restoration

4. eBPF Program Execution
   └─> On function entry: execute uprobe eBPF program
   └─> On function exit: execute uretprobe eBPF program
   └─> eBPF program can modify registers (arguments/return values)
```

#### 4. Configuration File Design and Interaction Modes

**Full Configuration Schema:**

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: advanced-fault-injection
  namespace: chaos-testing
spec:
  # Target Selection
  selector:
    namespaces:
      - production
    labelSelectors:
      app: payment-service
      tier: backend
    annotationSelectors:
      chaos.mesh.org/injectable: "true"
    podPhaseSelectors:
      - Running
  
  # Container targeting (optional, defaults to all containers)
  containerNames:
    - main-app
  
  # Execution mode
  mode: one  # Options: one, all, fixed, fixed-percent, random-max-percent
  value: "1"  # Number of pods to affect (for fixed mode)
  
  # Duration of chaos
  duration: "5m"
  
  # Scheduling (optional)
  scheduler:
    cron: "@every 30m"
  
  # Function Hook Configuration
  functionHook:
    # Target function
    function: "malloc"
    library: "libc.so.6"
    
    # Hook behavior
    action: fail  # Options: fail, delay, modifyReturn, modifyArgs
    
    # Fault injection parameters
    probability: 15  # 15% of calls will fail
    returnValue: "0"  # Return NULL
    errno: "ENOMEM"  # Set errno to ENOMEM (12)
    
    # Advanced filtering
    filter:
      # Only inject if malloc size > 1MB
      arguments:
        - index: 0  # First argument (size)
          greaterThan: 1048576
      
      # Only inject if called from specific functions
      callStack:
        - "largeAllocation"
        - "bufferResize"
    
    # Performance tuning
    maxEvents: 10000  # Stop after 10k injections
    cooldown: "1s"    # Minimum time between injections
  
  # Status tracking
  status:
    conditions: []
    experiment:
      desiredPhase: Running
```

**Interaction Modes:**

1. **Direct Mode** - Immediate effect
   ```yaml
   duration: "30s"  # Chaos starts immediately, lasts 30 seconds
   ```

2. **Scheduled Mode** - Periodic execution
   ```yaml
   scheduler:
     cron: "0 */2 * * *"  # Every 2 hours
   duration: "5m"
   ```

3. **Conditional Mode** - Triggered by metrics
   ```yaml
   # Future enhancement - trigger based on Prometheus metrics
   trigger:
     type: metric
     metric: "http_requests_per_second > 1000"
   ```

#### 5. Mapping bpftime Capabilities to Fault Scenarios

| Fault Scenario | bpftime Capability Used | Implementation Method | Reference |
|---------------|------------------------|---------------------|-----------|
| Memory Allocation Failures | Uprobe on malloc/calloc | Hook entry, check size, return NULL | [uprobe.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md) |
| File I/O Failures | Uprobe on fopen/fread | Hook entry, check path filter, return error | [example](https://github.com/eunomia-bpf/bpftime/tree/main/example/malloc) |
| Network Delays | Uprobe on connect/send | Hook entry, sleep for N ms, continue | [syscall-tracing.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/syscall-tracing.md) |
| Thread Failures | Uprobe on pthread_create | Hook entry, return EAGAIN with probability | [uprobe.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md) |
| SSL Failures | Uprobe on SSL_connect | Hook exit, modify return value to -1 | [uretprobe example](https://github.com/eunomia-bpf/bpftime/blob/main/example/uprobe) |
| Return Value Override | Uretprobe on any function | Hook exit, write to RAX/R0 register | [vm implementation](https://github.com/eunomia-bpf/bpftime/tree/main/vm) |
| Argument Modification | Uprobe with argument access | Read/write RDI, RSI, RDX registers | [eBPF helpers](https://github.com/eunomia-bpf/bpftime/blob/main/docs/available-features.md) |
| Call Stack Filtering | eBPF stack unwinding | Use `bpf_get_stackid()` helper | [maps documentation](https://ebpf.io/what-is-ebpf/#maps) |

**Example eBPF Program (pseudocode for malloc failure):**

```c
// Based on bpftime's eBPF program structure
// Reference: https://github.com/eunomia-bpf/bpftime/tree/main/example/malloc

#include <bpf/bpf_helpers.h>

// Configuration map (shared with userspace controller)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, struct fault_config);
    __uint(max_entries, 1024);
} config_map SEC(".maps");

struct fault_config {
    __u32 probability;  // 0-100
    __u64 min_size;     // Minimum allocation size to affect
    __u64 return_value; // Value to return (0 for NULL)
};

SEC("uprobe/malloc")
int handle_malloc(struct pt_regs *ctx)
{
    __u64 size = PT_REGS_PARM1(ctx);  // First parameter
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    struct fault_config *cfg = bpf_map_lookup_elem(&config_map, &pid);
    if (!cfg)
        return 0;  // No config, pass through
    
    // Filter by size
    if (size < cfg->min_size)
        return 0;
    
    // Probabilistic injection
    __u32 rand = bpf_get_prandom_u32() % 100;
    if (rand >= cfg->probability)
        return 0;
    
    // Override return value (requires uretprobe)
    bpf_override_return(ctx, cfg->return_value);
    
    // Set errno (requires syscall interception)
    // errno = ENOMEM;  // Implemented in userspace wrapper
    
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
```

### Architecture and Design

#### Component Architecture

```
┌─────────────────────────────────────────────────┐
│           Chaos Controller Manager               │
│  ┌───────────────────────────────────────────┐  │
│  │    UserspaceChaos Controller              │  │
│  └───────────────────────────────────────────┘  │
│                      │                           │
│                      ▼                           │
│  ┌───────────────────────────────────────────┐  │
│  │   UserspaceChaos Webhook & Validator      │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│              Chaos Daemon (DaemonSet)            │
│  ┌───────────────────────────────────────────┐  │
│  │        bpftime Runtime Manager            │  │
│  │  - Injects bpftime into target containers │  │
│  │  - Manages eBPF program lifecycle        │  │
│  │  - Handles hook attach/detach             │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│            Target Pod/Container                  │
│  ┌───────────────────────────────────────────┐  │
│  │         bpftime Agent (Injected)          │  │
│  │  - Loads eBPF programs                    │  │
│  │  - Hooks specified functions              │  │
│  │  - Executes fault injection logic         │  │
│  └───────────────────────────────────────────┘  │
│                      │                           │
│  ┌───────────────────────────────────────────┐  │
│  │      Target Application Process           │  │
│  │  - Instrumented by bpftime                │  │
│  │  - Functions hooked and faults injected   │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

#### API Specification

```go
// UserspaceChaos is the Schema for the userspacechaos API
type UserspaceChaos struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   UserspaceChaosSpec   `json:"spec"`
    Status UserspaceChaosStatus `json:"status,omitempty"`
}

// UserspaceChaosSpec defines the desired state of UserspaceChaos
type UserspaceChaosSpec struct {
    ContainerSelector `json:",inline"`
    
    // FunctionHook defines the function to hook and fault to inject
    FunctionHook FunctionHookSpec `json:"functionHook"`
    
    // Duration represents the duration of the chaos action
    Duration *string `json:"duration,omitempty"`
    
    // RemoteCluster represents the remote cluster where the chaos will be deployed
    RemoteCluster string `json:"remoteCluster,omitempty"`
}

// FunctionHookSpec defines what function to hook and how to inject faults
type FunctionHookSpec struct {
    // Function name to hook (supports mangled C++ names)
    Function string `json:"function"`
    
    // Library path (e.g., "libc.so.6", "libpthread.so.0")
    // If empty, hooks the main binary
    Library string `json:"library,omitempty"`
    
    // Binary path for application-specific function hooks
    Binary string `json:"binary,omitempty"`
    
    // Action defines the type of fault injection
    // +kubebuilder:validation:Enum=fail;delay;modifyReturn;modifyArgs
    Action FaultAction `json:"action"`
    
    // Probability of fault injection (0-100)
    // +kubebuilder:validation:Minimum=0
    // +kubebuilder:validation:Maximum=100
    Probability uint32 `json:"probability,omitempty"`
    
    // ReturnValue to set when action is fail or modifyReturn
    ReturnValue string `json:"returnValue,omitempty"`
    
    // Errno to set when action is fail (e.g., "ENOMEM", "ENOSPC")
    Errno string `json:"errno,omitempty"`
    
    // DelayMs introduces delay in milliseconds when action is delay
    DelayMs uint32 `json:"delayMs,omitempty"`
    
    // Filter defines conditions for when to apply the fault
    Filter *FunctionFilter `json:"filter,omitempty"`
}

// FaultAction defines the type of fault to inject
type FaultAction string

const (
    FaultActionFail         FaultAction = "fail"
    FaultActionDelay        FaultAction = "delay"
    FaultActionModifyReturn FaultAction = "modifyReturn"
    FaultActionModifyArgs   FaultAction = "modifyArgs"
)

// FunctionFilter defines filtering conditions
type FunctionFilter struct {
    // Arguments filtering based on parameter values
    Arguments []ArgumentFilter `json:"arguments,omitempty"`
    
    // CallStack filtering - only inject if specific function is in call stack
    CallStack []string `json:"callStack,omitempty"`
}

// ArgumentFilter defines how to filter based on function arguments
type ArgumentFilter struct {
    // Index of the argument (0-based)
    Index uint32 `json:"index"`
    
    // Contains string matching for string arguments
    Contains string `json:"contains,omitempty"`
    
    // Equals for exact matching
    Equals string `json:"equals,omitempty"`
    
    // GreaterThan for numeric arguments
    GreaterThan *int64 `json:"greaterThan,omitempty"`
    
    // LessThan for numeric arguments
    LessThan *int64 `json:"lessThan,omitempty"`
}
```

### Technical Implementation Details

This section provides detailed technical explanations of how bpftime enables each fault injection scenario, with references to specific bpftime capabilities.

#### Detailed Scenario Implementation

##### 1. Memory Allocation Failures - Technical Flow

**bpftime Capabilities Used:**
- Uprobe attachment ([uprobe.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md))
- Uretprobe for return value modification ([example](https://github.com/eunomia-bpf/bpftime/tree/main/example/malloc))
- eBPF maps for configuration storage

**Implementation Steps:**
1. Attach uprobe to `malloc` entry point in libc.so.6
2. eBPF program reads requested size from RDI register (x86_64 calling convention)
3. Check configuration map for probability and size filters
4. If fault should be injected:
   - Skip original malloc call using `bpf_override_return()`
   - Return NULL (0x0) via RAX register
   - Set thread-local errno to ENOMEM via syscall interception
5. If fault not injected, allow original malloc to execute

**Code Flow:**
```
User YAML → Controller → Daemon → bpftime
                                     ↓
                              Load eBPF program
                                     ↓
                              Attach uprobe to malloc
                                     ↓
                     malloc() called in app
                                     ↓
                     eBPF program executes
                                     ↓
              Check probability (using bpf_get_prandom_u32)
                                     ↓
                      [Inject fault? Yes/No]
                                     ↓
              Yes: Override return → NULL
              No: Continue to real malloc
```

##### 2. File I/O Failures - Argument Filtering

**bpftime Capabilities Used:**
- Uprobe with argument reading ([PT_REGS_PARM macros](https://github.com/eunomia-bpf/bpftime/blob/main/docs/available-features.md))
- String comparison in eBPF
- Return value override

**Path Filtering Implementation:**
```c
SEC("uprobe/fopen")
int handle_fopen(struct pt_regs *ctx)
{
    // Read first argument (filename pointer)
    const char *filename = (const char *)PT_REGS_PARM1(ctx);
    
    // Read filename into eBPF (max 256 bytes)
    char path[256];
    bpf_probe_read_user_str(path, sizeof(path), filename);
    
    // Check filter: only affect /data/ paths
    if (bpf_strstr(path, "/data/") == NULL)
        return 0;  // No match, pass through
    
    // Apply probabilistic fault injection
    if (should_inject_fault()) {
        // Override return to NULL (file open failed)
        bpf_override_return(ctx, 0);
        // errno set via userspace wrapper
    }
    
    return 0;
}
```

**Reference:** [bpf_probe_read_user_str helper](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html)

##### 3. Network Delays - Sleep Injection

**bpftime Capabilities Used:**
- Uprobe for function entry interception
- Userspace sleep via eBPF program (using `usleep` syscall wrapper)
- Minimal overhead via JIT compilation

**Delay Implementation:**
```c
SEC("uprobe/connect")
int handle_connect(struct pt_regs *ctx)
{
    struct delay_config *cfg = get_config();
    if (!cfg || !should_inject_fault())
        return 0;
    
    // Inject delay by calling usleep
    // bpftime allows eBPF programs to invoke userspace functions
    usleep(cfg->delay_ms * 1000);
    
    // Continue to real connect() call
    return 0;
}
```

**Performance Note:** Based on [bpftime benchmarks](https://github.com/eunomia-bpf/bpftime#performance), uprobe overhead is <5% for most operations, making delay injection accurate.

##### 4. Thread Creation Failures - Error Code Injection

**bpftime Capabilities Used:**
- Uretprobe for return value modification
- Return value override via register manipulation

**pthread_create Failure Implementation:**
```c
SEC("uretprobe/pthread_create")
int handle_pthread_create_exit(struct pt_regs *ctx)
{
    if (!should_inject_fault())
        return 0;
    
    // Override return value to EAGAIN (11)
    // RAX register holds return value on x86_64
    PT_REGS_RC(ctx) = 11;  // EAGAIN
    
    // Log the injection for observability
    struct event e = {
        .pid = bpf_get_current_pid_tgid(),
        .timestamp = bpf_ktime_get_ns(),
        .fault_type = FAULT_PTHREAD_CREATE,
    };
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &e, sizeof(e));
    
    return 0;
}
```

##### 5. Call Stack Filtering - Advanced Feature

**bpftime Capabilities Used:**
- Stack unwinding via `bpf_get_stackid()` ([helper documentation](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html))
- Stack map storage

**Call Stack Check Implementation:**
```c
// Stack trace map
struct {
    __uint(type, BPF_MAP_TYPE_STACK_TRACE);
    __uint(max_entries, 1024);
} stack_traces SEC(".maps");

SEC("uprobe/malloc")
int handle_malloc_with_stack_filter(struct pt_regs *ctx)
{
    // Get current stack trace
    int stack_id = bpf_get_stackid(ctx, &stack_traces, BPF_F_USER_STACK);
    if (stack_id < 0)
        return 0;
    
    // Check if specific function is in call stack
    // In practice, this requires symbol resolution
    if (!is_function_in_stack(stack_id, "largeAllocation"))
        return 0;
    
    // Inject fault only if called from largeAllocation()
    return inject_malloc_failure(ctx);
}
```

**Reference:** [bpf_get_stackid documentation](https://github.com/iovisor/bcc/blob/master/docs/reference_guide.md#4-bpf_get_stackid)

#### Integration with Chaos Daemon

**Daemon Component Responsibilities:**

1. **bpftime Runtime Management**
   - Download and cache bpftime binary
   - Inject bpftime into target container via nsenter
   - Manage bpftime process lifecycle

2. **eBPF Program Compilation**
   - Convert YAML configuration to eBPF C code
   - Compile using clang/LLVM to eBPF bytecode
   - Load into bpftime runtime

3. **Hook Lifecycle Management**
   ```go
   // Pseudo-code for daemon integration
   func (d *Daemon) ApplyUserspaceChaos(chaos *UserspaceChaos) error {
       // 1. Resolve target container PID
       pid := d.GetContainerPID(chaos.Spec.ContainerName)
       
       // 2. Generate eBPF program
       ebpfProg := d.GenerateEBPFProgram(chaos.Spec.FunctionHook)
       
       // 3. Inject bpftime into target namespace
       bpftimeCmd := fmt.Sprintf(
           "nsenter -t %d -p -m -- /usr/local/bin/bpftime load %s",
           pid, ebpfProg,
       )
       
       // 4. Attach hooks
       err := d.ExecuteInNamespace(pid, bpftimeCmd)
       if err != nil {
           return fmt.Errorf("failed to attach hooks: %w", err)
       }
       
       // 5. Monitor hook status
       d.MonitorHooks(chaos.Name, pid)
       
       return nil
   }
   ```

**Reference:** Similar to [KernelChaos implementation](https://github.com/chaos-mesh/chaos-mesh/blob/master/controllers/chaosimpl/kernelchaos/types.go)

#### Configuration Validation

**Webhook Validation Rules:**

```go
// Validation ensures configuration is correct before deployment
func (webhook *UserspaceChaosWebhook) ValidateCreate(obj runtime.Object) error {
    chaos := obj.(*UserspaceChaos)
    
    // 1. Validate function exists (pre-flight check)
    if chaos.Spec.FunctionHook.Library != "" {
        if !isLibraryAvailable(chaos.Spec.FunctionHook.Library) {
            return fmt.Errorf("library not found: %s", chaos.Spec.FunctionHook.Library)
        }
    }
    
    // 2. Validate action parameters
    switch chaos.Spec.FunctionHook.Action {
    case FaultActionFail, FaultActionModifyReturn:
        if chaos.Spec.FunctionHook.ReturnValue == "" {
            return fmt.Errorf("returnValue required for action: %s", chaos.Spec.FunctionHook.Action)
        }
    case FaultActionDelay:
        if chaos.Spec.FunctionHook.DelayMs == 0 {
            return fmt.Errorf("delayMs required for action: delay")
        }
    }
    
    // 3. Validate probability range
    if chaos.Spec.FunctionHook.Probability > 100 {
        return fmt.Errorf("probability must be 0-100, got: %d", chaos.Spec.FunctionHook.Probability)
    }
    
    return nil
}
```

### Implementation Plan

#### Phase 1: Core Infrastructure (Weeks 1-3)
1. **bpftime Integration**
   - Package bpftime binary with Chaos Daemon image
   - Create injection mechanism to load bpftime into target containers
   - Implement gRPC protocol between Controller and Daemon for bpftime operations

2. **CRD and API Definition**
   - Define UserspaceChaos CRD schema
   - Implement validation webhooks
   - Add to API group v1alpha1

3. **Basic Controller Implementation**
   - Create UserspaceChaos controller
   - Implement Apply and Recover methods
   - Handle selector logic for pod targeting

#### Phase 2: Fault Injection Capabilities (Weeks 4-6)
1. **eBPF Program Templates**
   - Create reusable eBPF programs for common fault scenarios
   - Implement fail action (return value modification)
   - Implement delay action (sleep injection)

2. **Daemon Integration**
   - Extend Chaos Daemon to manage bpftime lifecycle
   - Implement function hook attachment/detachment
   - Add monitoring and logging for injected faults

3. **Testing Framework**
   - Create e2e tests for basic scenarios
   - Test with common libraries (libc, libpthread)
   - Validate fault injection accuracy

#### Phase 3: Advanced Features (Weeks 7-9)
1. **Filtering and Conditions**
   - Implement argument-based filtering
   - Add call stack filtering
   - Support complex condition expressions

2. **Library-Specific Helpers**
   - Pre-built configurations for common libraries
   - Document common use cases
   - Create example yamls

3. **Observability**
   - Add metrics for injection success/failure rates
   - Implement tracing for hooked function calls
   - Dashboard visualization support

#### Phase 4: Documentation and Stabilization (Weeks 10-12)
1. **Documentation**
   - API reference documentation
   - User guide with examples
   - Best practices and troubleshooting

2. **Performance Optimization**
   - Minimize overhead of bpftime injection
   - Optimize eBPF program efficiency
   - Load testing and benchmarking

3. **Security Hardening**
   - RBAC configuration guidelines
   - Container security context requirements
   - Audit logging for fault injections

### Security Considerations

1. **Privilege Requirements**
   - bpftime runs in userspace but requires `CAP_SYS_PTRACE` to attach to processes
   - May need `CAP_SYS_ADMIN` for certain syscall interception scenarios
   - Should document minimum required capabilities

2. **Isolation**
   - Injected faults are isolated to target containers
   - No impact on other pods on the same node
   - Proper cleanup on chaos recovery

3. **Access Control**
   - RBAC policies to control who can create UserspaceChaos
   - Namespace isolation enforced
   - Audit trail for all chaos operations

### Advantages Over Existing Solutions

1. **No Kernel Modifications Required**
   - Unlike KernelChaos, doesn't require kernel module support
   - Works on any Linux kernel version
   - No risk of kernel instability

2. **Application-Level Precision**
   - Can target specific functions in specific binaries
   - Filter by argument values
   - More granular than network or IO chaos

3. **Performance**
   - Lower overhead than kernel-based tracing
   - JIT compilation for eBPF programs
   - Minimal impact on non-hooked code paths

4. **Flexibility**
   - Can hook any userspace function
   - Support for compiled binaries without source code
   - Works with C, C++, Go, Rust applications

### Limitations and Trade-offs

1. **Language Support**
   - Best support for C/C++ applications
   - Limited for interpreted languages (Python, Ruby)
   - Go support depends on symbol availability (may need -ldflags for symbols)

2. **Container Compatibility**
   - Requires process namespace access
   - May have issues with minimal/distroless containers
   - Needs specific libraries present in container

3. **Complexity**
   - More complex setup than other chaos types
   - Requires understanding of function symbols and calling conventions
   - Debugging can be challenging

4. **Overhead**
   - While minimal, there is still overhead from instrumentation
   - Not suitable for extreme performance-critical paths
   - May affect latency-sensitive applications

### Alternatives Considered

1. **LD_PRELOAD-based Injection**
   - Pros: Simpler, well-understood mechanism
   - Cons: Limited to dynamic library functions, requires application restart, easier to bypass

2. **GDB/PTRACE-based Injection**
   - Pros: Can modify any running process
   - Cons: Very high overhead, pauses application, not suitable for production

3. **Extend KernelChaos with Uprobe**
   - Pros: Reuses existing infrastructure
   - Cons: Still requires kernel support, higher overhead, less flexible

4. **Custom Application Instrumentation**
   - Pros: Most accurate and efficient
   - Cons: Requires code changes, not applicable to third-party software

### Success Metrics

1. **Functionality**
   - Support hooking at least 20 common libc functions
   - Successfully inject faults in 3+ different application types (C, C++, Go)
   - <5% false negative rate (faults not injected when should be)

2. **Performance**
   - <10% performance overhead on hooked functions
   - <1% overhead on non-hooked code paths
   - bpftime injection completes in <5 seconds

3. **Usability**
   - Documentation covers 10+ real-world scenarios
   - 90%+ of users can successfully create UserspaceChaos in first attempt
   - Clear error messages for misconfigurations

4. **Reliability**
   - 99.9% successful recovery (no hanging processes)
   - Zero kernel panics or system crashes
   - All resources properly cleaned up after chaos ends

### Future Enhancements

1. **Advanced eBPF Programs**
   - Support custom eBPF programs provided by users
   - State tracking across multiple function calls
   - Complex fault injection patterns (e.g., "fail every 3rd call")

2. **Integration with Observability**
   - Automatic correlation with metrics/logs
   - Distributed tracing integration
   - Fault injection visualization

3. **Multi-Language Support**
   - Better support for Go applications
   - JVM integration via JVMTI
   - Python/Node.js native module hooking

4. **Intelligent Fault Generation**
   - ML-based fault scenario generation
   - Automatic discovery of critical functions
   - Coverage-guided fault injection

## References

### Primary Sources

1. **bpftime Project**
   - [bpftime GitHub Repository](https://github.com/eunomia-bpf/bpftime) - Main project repository
   - [bpftime Technical Paper](https://arxiv.org/abs/2311.07923) - "bpftime: Userspace eBPF Runtime for Fast Uprobes"
   - [bpftime Documentation](https://github.com/eunomia-bpf/bpftime/tree/main/docs) - Technical documentation and guides
   - [Uprobe Implementation](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md) - Uprobe/uretprobe support details
   - [Syscall Tracing Guide](https://github.com/eunomia-bpf/bpftime/blob/main/docs/syscall-tracing.md) - Syscall interception methodology

2. **eBPF Foundation**
   - [eBPF Official Website](https://ebpf.io/) - What is eBPF and core concepts
   - [eBPF Documentation](https://ebpf.io/what-is-ebpf/) - Maps, helpers, and program types
   - [BPF and XDP Reference Guide](https://docs.cilium.io/en/latest/bpf/) - Comprehensive BPF reference
   - [Linux eBPF Helpers](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html) - Available BPF helper functions

3. **Linux Kernel Documentation**
   - [Linux Uprobe Documentation](https://www.kernel.org/doc/html/latest/trace/uprobetracer.html) - Kernel uprobe tracing
   - [Linux Tracing Technologies](https://www.kernel.org/doc/html/latest/trace/ftrace.html) - ftrace and related tracing systems
   - [BPF Design Q&A](https://www.kernel.org/doc/html/latest/bpf/bpf_design_QA.html) - BPF design decisions

4. **Chaos Mesh Project**
   - [Chaos Mesh Documentation](https://chaos-mesh.org/) - Official documentation
   - [KernelChaos Design](https://chaos-mesh.org/docs/simulate-kernel-chaos/) - Existing kernel-level fault injection
   - [Chaos Mesh Architecture](https://chaos-mesh.org/docs/basic-features/) - System architecture overview
   - [Development Guide](https://chaos-mesh.org/docs/developer-guide-overview/) - Developer resources

### Implementation Examples

5. **bpftime Examples**
   - [malloc Hook Example](https://github.com/eunomia-bpf/bpftime/tree/main/example/malloc) - Memory allocation hooking
   - [Uprobe Examples](https://github.com/eunomia-bpf/bpftime/tree/main/example/uprobe) - Function hooking samples
   - [Runtime Implementation](https://github.com/eunomia-bpf/bpftime/tree/main/runtime) - Core runtime code
   - [VM Implementation](https://github.com/eunomia-bpf/bpftime/tree/main/vm) - eBPF virtual machine and JIT

6. **Related Projects and Tools**
   - [Frida](https://frida.re/) - Dynamic instrumentation toolkit (used by bpftime)
   - [libbpf](https://github.com/libbpf/libbpf) - eBPF library for Linux
   - [BCC Tools](https://github.com/iovisor/bcc) - BPF Compiler Collection
   - [bpftrace](https://github.com/iovisor/bpftrace) - High-level tracing language

### Academic and Technical Papers

7. **Research Papers**
   - [bpftime: Userspace eBPF Runtime for Fast Uprobes](https://arxiv.org/abs/2311.07923) - Core technical paper
   - [eBPF: A New Approach to Cloud-Native Observability](https://dl.acm.org/doi/10.1145/3544497.3544498) - eBPF applications
   - [Chaos Engineering at Scale](https://queue.acm.org/detail.cfm?id=2353017) - Netflix's chaos engineering principles

### Specifications and Standards

8. **ELF and Symbol Format**
   - [ELF Format Specification](https://refspecs.linuxfoundation.org/elf/elf.pdf) - Executable and Linkable Format
   - [DWARF Debugging Standard](https://dwarfstd.org/) - Debugging information format
   - [System V ABI](https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf) - Application Binary Interface for x86_64

9. **Container and Kubernetes**
   - [Kubernetes CRD Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/) - Custom Resource Definitions
   - [Container Runtime Interface](https://github.com/kubernetes/cri-api) - CRI specification
   - [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html) - Process isolation mechanisms

## Conclusion

Adding UserspaceChaos to Chaos Mesh through bpftime integration will significantly expand fault injection capabilities, enabling application-level chaos engineering that was previously difficult or impossible. This will help users:

- Test error handling in their applications more thoroughly
- Verify resilience of third-party library integration
- Simulate real-world failures at the function call level
- Improve overall system reliability through more comprehensive testing

The proposed implementation is feasible, leverages proven technology (eBPF and bpftime), and aligns with Chaos Mesh's goal of providing comprehensive chaos engineering capabilities for cloud-native applications.
