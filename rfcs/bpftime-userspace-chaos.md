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

1. [bpftime GitHub Repository](https://github.com/eunomia-bpf/bpftime)
2. [eBPF Documentation](https://ebpf.io/)
3. [Linux Uprobe Documentation](https://www.kernel.org/doc/html/latest/trace/uprobetracer.html)
4. [Chaos Mesh Documentation](https://chaos-mesh.org/)
5. [KernelChaos Design](https://chaos-mesh.org/docs/simulate-kernel-chaos/)

## Conclusion

Adding UserspaceChaos to Chaos Mesh through bpftime integration will significantly expand fault injection capabilities, enabling application-level chaos engineering that was previously difficult or impossible. This will help users:

- Test error handling in their applications more thoroughly
- Verify resilience of third-party library integration
- Simulate real-world failures at the function call level
- Improve overall system reliability through more comprehensive testing

The proposed implementation is feasible, leverages proven technology (eBPF and bpftime), and aligns with Chaos Mesh's goal of providing comprehensive chaos engineering capabilities for cloud-native applications.
