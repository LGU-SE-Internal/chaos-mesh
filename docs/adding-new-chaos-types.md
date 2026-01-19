# Adding New Chaos Types to Chaos Mesh

This guide provides a comprehensive walkthrough for implementing new chaos injection types (controllers) in Chaos Mesh. It covers the complete workflow from development to testing and verification.

## Table of Contents

1. [Prerequisites and Requirements](#prerequisites-and-requirements)
2. [Architecture Overview](#architecture-overview)
3. [Step-by-Step Implementation Guide](#step-by-step-implementation-guide)
4. [Code Generation](#code-generation)
5. [Testing](#testing)
6. [Best Practices](#best-practices)
7. [Complete Checklist](#complete-checklist)
8. [Troubleshooting](#troubleshooting)

## Prerequisites and Requirements

Before implementing a new chaos type, ensure you have:

- **Go programming knowledge**: Familiarity with Go 1.19+ and common patterns
- **Kubernetes understanding**: Knowledge of CRDs, controllers, and the controller-runtime framework
- **Chaos Mesh architecture**: Understanding of the controller-manager and daemon architecture
- **Development environment**: A working Kubernetes cluster for testing

### Key Concepts

- **CRD (Custom Resource Definition)**: Defines the schema for your chaos type
- **Controller**: Reconciles the desired state with actual state
- **Chaos Daemon**: Privileged DaemonSet that performs actual fault injection
- **Actions**: Different fault injection modes within a chaos type (e.g., pod-kill, pod-failure)

## Architecture Overview

A chaos type in Chaos Mesh consists of several interconnected components:

```
┌─────────────────────────────────────────────────────────────┐
│ api/v1alpha1/*chaos_types.go                                │
│ - CRD type definition with kubebuilder annotations          │
│ - Spec and Status structs                                   │
│ - Action constants                                          │
│ - Interface implementations (InnerObject, etc.)             │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ api/v1alpha1/*chaos_webhook.go                              │
│ - Webhook validation logic                                  │
│ - Validate() method implementation                          │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ controllers/chaosimpl/*chaos/                               │
│ - Controller implementation with fx dependency injection    │
│ - Action-based implementations (Apply/Recover methods)      │
│ - Multiplexer for routing actions                           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/chaosdaemon/ (optional)                                 │
│ - gRPC service definitions in protobuf                      │
│ - Server implementations for daemon-level operations        │
└─────────────────────────────────────────────────────────────┘
```

## Step-by-Step Implementation Guide

### Step 1: Define CRD Type

Create a new file `api/v1alpha1/<chaostype>_types.go` that defines your chaos type's structure.

**Example from PodChaos:**

```go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +chaos-mesh:experiment
// +chaos-mesh:oneshot=in.Spec.Action==PodKillAction || in.Spec.Action==ContainerKillAction
// +genclient

// PodChaos is the control script's spec.
type PodChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of a pod chaos experiment
	Spec PodChaosSpec `json:"spec"`

	// +optional
	// Most recently observed status of the chaos experiment about pods
	Status PodChaosStatus `json:"status,omitempty"`
}

var _ InnerObjectWithSelector = (*PodChaos)(nil)
var _ InnerObject = (*PodChaos)(nil)

// PodChaosAction represents the chaos action about pods.
type PodChaosAction string

const (
	// PodKillAction represents the chaos action of killing pods.
	PodKillAction PodChaosAction = "pod-kill"
	// PodFailureAction represents the chaos action of injecting errors to pods.
	PodFailureAction PodChaosAction = "pod-failure"
	// ContainerKillAction represents the chaos action of killing the container
	ContainerKillAction PodChaosAction = "container-kill"
)

// PodChaosSpec defines the attributes that a user creates on a chaos experiment about pods.
type PodChaosSpec struct {
	ContainerSelector `json:",inline"`

	// Action defines the specific pod chaos action.
	// Supported action: pod-kill / pod-failure / container-kill
	// Default action: pod-kill
	// +kubebuilder:validation:Enum=pod-kill;pod-failure;container-kill
	Action PodChaosAction `json:"action"`

	// Duration represents the duration of the chaos action.
	// +optional
	Duration *string `json:"duration,omitempty" webhook:"Duration"`

	// GracePeriod is used in pod-kill action.
	// +optional
	// +kubebuilder:validation:Minimum=0
	GracePeriod int64 `json:"gracePeriod,omitempty"`

	// RemoteCluster represents the remote cluster where the chaos will be deployed
	// +optional
	RemoteCluster string `json:"remoteCluster,omitempty"`
}

// PodChaosStatus represents the current status of the chaos experiment about pods.
type PodChaosStatus struct {
	ChaosStatus `json:",inline"`
}

func (obj *PodChaos) GetSelectorSpecs() map[string]interface{} {
	switch obj.Spec.Action {
	case PodKillAction, PodFailureAction:
		return map[string]interface{}{
			".": &obj.Spec.PodSelector,
		}
	case ContainerKillAction:
		return map[string]interface{}{
			".": &obj.Spec.ContainerSelector,
		}
	}
	return nil
}
```

**Key annotations:**

- `+kubebuilder:object:root=true`: Marks this as a root Kubernetes object
- `+chaos-mesh:experiment`: Registers this as a chaos experiment type
- `+chaos-mesh:oneshot=<condition>`: Defines when this is a one-shot experiment (no recovery needed)
- `+genclient`: Generates client code for this type
- `+kubebuilder:validation:Enum=...`: Validates enum values
- `+kubebuilder:validation:Minimum=N`: Validates minimum numeric values

**Required interface implementations:**

```go
var _ InnerObjectWithSelector = (*YourChaos)(nil)
var _ InnerObject = (*YourChaos)(nil)
```

### Step 2: Implement Webhook Validation

Create `api/v1alpha1/<chaostype>_webhook.go` to validate user input.

**Example from PodChaos:**

```go
package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate validates the PodChaosSpec
func (in *PodChaosSpec) Validate(root interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if in.Action == ContainerKillAction {
		if len(in.ContainerSelector.ContainerNames) == 0 {
			err := errors.Wrapf(errInvalidValue, "the name of container is required on %s action", in.Action)
			allErrs = append(allErrs, field.Invalid(path.Child("containerNames"), in.ContainerNames, err.Error()))
		}
	}
	return allErrs
}
```

**Validation patterns:**

- Return `field.ErrorList` with all validation errors
- Use `field.Invalid()`, `field.Required()`, `field.Forbidden()` for different error types
- Validate action-specific requirements
- Check for required fields based on selected action

### Step 3: Create Controller Implementation

Create a directory structure under `controllers/chaosimpl/<chaostype>/` with the following files:

#### Main Implementation File: `impl.go`

This file aggregates all action implementations using fx dependency injection.

**Example from PodChaos:**

```go
package podchaos

import (
	"go.uber.org/fx"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/controllers/action"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/podchaos/containerkill"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/podchaos/podfailure"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/podchaos/podkill"
	impltypes "github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/types"
)

type Impl struct {
	fx.In

	PodKill       *podkill.Impl       `action:"pod-kill"`
	PodFailure    *podfailure.Impl    `action:"pod-failure"`
	ContainerKill *containerkill.Impl `action:"container-kill"`
}

func NewImpl(impl Impl) *impltypes.ChaosImplPair {
	delegate := action.NewMultiplexer(&impl)
	return &impltypes.ChaosImplPair{
		Name:   "podchaos",
		Object: &v1alpha1.PodChaos{},
		Impl:   &delegate,
	}
}

var Module = fx.Provide(
	fx.Annotated{
		Group:  "impl",
		Target: NewImpl,
	},
	podkill.NewImpl,
	podfailure.NewImpl,
	containerkill.NewImpl,
)
```

**Key patterns:**

- Use `fx.In` for dependency injection
- Tag each action implementation with `action:"action-name"`
- Use `action.NewMultiplexer()` to route actions
- Export an `fx.Module` for registration

#### Action Implementation Files: `<action>/impl.go`

Each action needs its own implementation that satisfies the `ChaosImpl` interface.

**Example from PodKill:**

```go
package podkill

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	impltypes "github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/types"
	"github.com/chaos-mesh/chaos-mesh/controllers/utils/controller"
)

var _ impltypes.ChaosImpl = (*Impl)(nil)

type Impl struct {
	client.Client
}

func (impl *Impl) Apply(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	podchaos := obj.(*v1alpha1.PodChaos)

	var pod v1.Pod
	namespacedName, err := controller.ParseNamespacedName(records[index].Id)
	if err != nil {
		return v1alpha1.NotInjected, err
	}
	err = impl.Get(ctx, namespacedName, &pod)
	if err != nil {
		return v1alpha1.NotInjected, err
	}

	err = impl.Delete(ctx, &pod, &client.DeleteOptions{
		GracePeriodSeconds: &podchaos.Spec.GracePeriod,
	})
	if err != nil {
		return v1alpha1.NotInjected, err
	}

	return v1alpha1.Injected, nil
}

func (impl *Impl) Recover(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	// Pod kill is a one-shot action, no recovery needed
	return v1alpha1.NotInjected, nil
}

func NewImpl(c client.Client) *Impl {
	return &Impl{
		Client: c,
	}
}
```

**ChaosImpl interface:**

```go
type ChaosImpl interface {
	Apply(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error)
	Recover(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error)
}
```

**Implementation guidelines:**

- `Apply()`: Injects the fault for a specific target (identified by `records[index]`)
- `Recover()`: Removes the fault and restores normal operation
- Return `v1alpha1.Injected` on successful injection, `v1alpha1.NotInjected` otherwise
- For one-shot actions (like pod-kill), `Recover()` can return immediately
- Use type assertion to convert `obj` to your specific chaos type

### Step 4: Add Daemon Implementation (Optional)

If your chaos type requires privileged operations or needs to interact with the host system, implement gRPC services in the chaos daemon.

**Location:** `pkg/chaosdaemon/`

1. Define gRPC service in `pb/chaosdaemon.proto`
2. Implement server methods in appropriate files
3. Register the service in the daemon server

**Example use cases:**

- Network manipulation (tc, iptables)
- Filesystem operations
- Process manipulation
- Time skewing
- Kernel-level operations

### Step 5: Register in All Required Locations

#### A. Register in `controllers/types/types.go`

Add your chaos type to the `ChaosObjects` list:

```go
var ChaosObjects = fx.Supply(
	// ... existing entries ...

	fx.Annotated{
		Group: "objs",
		Target: Object{
			Name:   "yourchaos",
			Object: &v1alpha1.YourChaos{},
		},
	},
)
```

#### B. Register in `controllers/chaosimpl/fx.go`

Add your module to the `AllImpl` options:

```go
var AllImpl = fx.Options(
	// ... existing modules ...
	yourchaos.Module,
	// ...
)
```

Don't forget to import your package:

```go
import (
	// ... existing imports ...
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/yourchaos"
)
```

#### C. Update Makefile (if needed)

If your chaos type has a non-standard plural form, add it to the `CHAOS_GROUP` variable in the Makefile:

```makefile
CHAOS_GROUP := \
	podchaos:podchaos \
	httpchaos:httpchaos \
	yourchaos:yourchaos
```

**Standard pluralization:** Most chaos types follow standard Go pluralization (add 's'). Only add to Makefile if your plural is non-standard.

## Code Generation

After implementing all components, regenerate the derived code:

```bash
# Generate all code (CRD manifests, deepcopy, client code, swagger spec)
make generate

# Verify formatting and linting
make check
```

**What gets generated:**

- **CRD manifests** (`manifests/crd.yaml`): Kubernetes CRD definitions
- **Deepcopy methods** (`zz_generated.deepcopy.go`): Required for Kubernetes objects
- **Client code** (`pkg/client/`): Clientset, informers, and listers
- **Swagger spec** (`api/openapi-spec/swagger.yaml`): API documentation

**Common generation errors:**

- Missing kubebuilder annotations
- Invalid validation rules
- Incorrect interface implementations
- Missing imports

## Testing

### Unit Testing

Create unit tests in your implementation package:

**Location:** `controllers/chaosimpl/<chaostype>/<action>/impl_test.go`

**Test patterns:**

```go
func TestApply(t *testing.T) {
	// Setup test environment
	// Create mock objects
	// Call Apply()
	// Verify expected behavior
}

func TestRecover(t *testing.T) {
	// Setup test environment
	// Create mock objects
	// Call Recover()
	// Verify cleanup
}
```

Run unit tests:

```bash
# Run all tests
make test

# Run tests for specific package
go test ./controllers/chaosimpl/yourchaos/... -v
```

### E2E Testing

Create end-to-end tests to verify the complete workflow in a real Kubernetes cluster.

**Location:** `e2e-test/e2e/chaos/<chaostype>/`

**Example from PodChaos:**

```go
package yourchaos

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/e2e-test/pkg/fixture"
)

func TestcaseYourChaosBasic(ns string, kubeCli kubernetes.Interface, cli client.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create target workload
	pod := fixture.NewCommonNginxPod("nginx", ns)
	_, err := kubeCli.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	framework.ExpectNoError(err, "create nginx pod error")

	// Create chaos experiment
	chaos := &v1alpha1.YourChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-chaos",
			Namespace: ns,
		},
		Spec: v1alpha1.YourChaosSpec{
			Action: v1alpha1.YourAction,
			// ... configure spec ...
		},
	}
	err = cli.Create(ctx, chaos)
	framework.ExpectNoError(err, "create chaos error")

	// Verify chaos effect
	// ... add verification logic ...

	// Cleanup
	err = cli.Delete(ctx, chaos)
	framework.ExpectNoError(err, "delete chaos error")
}
```

**E2E test patterns:**

1. Create target workload (pods, deployments, services)
2. Create chaos experiment CR
3. Wait for chaos to take effect
4. Verify expected behavior (pod killed, network delayed, etc.)
5. Pause/unpause testing (optional)
6. Cleanup resources

Run E2E tests:

```bash
# Build e2e test binaries
make e2e-build

# Run e2e tests (requires running Kubernetes cluster)
make e2e
```

### Manual Verification

1. **Deploy Chaos Mesh:**
   ```bash
   helm install chaos-mesh helm/chaos-mesh -n chaos-mesh --create-namespace
   ```

2. **Create a test workload:**
   ```bash
   kubectl create deployment nginx --image=nginx --replicas=3
   ```

3. **Apply your chaos experiment:**
   ```yaml
   apiVersion: chaos-mesh.org/v1alpha1
   kind: YourChaos
   metadata:
     name: test-chaos
   spec:
     action: your-action
     selector:
       namespaces:
         - default
       labelSelectors:
         app: nginx
     mode: one
   ```

4. **Verify the chaos effect:**
   ```bash
   kubectl get pods -w
   kubectl describe yourchaos test-chaos
   ```

5. **Check controller logs:**
   ```bash
   kubectl logs -n chaos-mesh -l app.kubernetes.io/component=controller-manager -f
   ```

## Best Practices

### Controller Design Principles

From `controllers/README.md`:

1. **One controller per field**: Each field should be controlled by at most one controller to avoid conflicts
2. **Standalone operation**: Controllers should work independently without dependencies on other controllers
3. **Error handling**: Return `ctrl.Result{Requeue: true}, nil` for retriable errors to leverage exponential backoff

### Error Handling Patterns

**Retriable errors:**
```go
if err != nil {
	// Transient error, will retry with exponential backoff
	return v1alpha1.NotInjected, err
}
```

**Non-retriable errors:**
```go
if err != nil {
	// Permanent error, log and don't retry
	log.Error(err, "permanent error occurred")
	return v1alpha1.NotInjected, nil
}
```

### Resource Cleanup

Always implement proper cleanup in the `Recover()` method:

```go
func (impl *Impl) Recover(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	// Remove injected faults
	// Restore original state
	// Clean up temporary resources

	return v1alpha1.NotInjected, nil
}
```

### Action Naming Conventions

- Use kebab-case for action names: `pod-kill`, `network-delay`, `io-errno`
- Action names should be descriptive and concise
- Group related actions under the same chaos type

### Validation Best Practices

- Validate all required fields in the webhook
- Provide clear error messages
- Validate action-specific requirements
- Check for conflicting configurations

## Complete Checklist

Use this checklist to ensure you've completed all necessary steps:

- [ ] **CRD Type Definition** (`api/v1alpha1/<chaostype>_types.go`)
  - [ ] Main struct with kubebuilder annotations
  - [ ] Spec struct with validation annotations
  - [ ] Status struct
  - [ ] Action constants defined
  - [ ] Interface implementations (InnerObject, InnerObjectWithSelector)
  - [ ] GetSelectorSpecs() method implemented

- [ ] **Webhook Validation** (`api/v1alpha1/<chaostype>_webhook.go`)
  - [ ] Validate() method implemented
  - [ ] Action-specific validation logic
  - [ ] Clear error messages

- [ ] **Controller Implementation** (`controllers/chaosimpl/<chaostype>/`)
  - [ ] Main impl.go with fx module
  - [ ] Action implementations with Apply/Recover methods
  - [ ] Multiplexer setup for routing actions
  - [ ] NewImpl constructor functions

- [ ] **Daemon Implementation** (if needed) (`pkg/chaosdaemon/`)
  - [ ] gRPC service definition in protobuf
  - [ ] Server method implementations
  - [ ] Service registration

- [ ] **Registration**
  - [ ] Added to `controllers/types/types.go` (ChaosObjects)
  - [ ] Added to `controllers/chaosimpl/fx.go` (AllImpl)
  - [ ] Makefile plural exceptions (if non-standard)

- [ ] **Code Generation**
  - [ ] Run `make generate` successfully
  - [ ] Run `make check` without errors
  - [ ] Verify generated CRD manifests

- [ ] **Testing**
  - [ ] Unit tests for each action
  - [ ] E2E tests for complete workflow
  - [ ] Manual verification in test cluster

- [ ] **Documentation**
  - [ ] Update relevant documentation
  - [ ] Add examples to docs/
  - [ ] Update CLAUDE.md if needed

## Troubleshooting

### Code Generation Errors

**Error: "no matches for kind YourChaos"**
- Ensure `+kubebuilder:object:root=true` annotation is present
- Run `make generate` to regenerate CRD manifests

**Error: "undefined: InnerObject"**
- Add import: `"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"`
- Verify interface implementations

**Error: "deepcopy method not found"**
- Run `make generate-deepcopy`
- Ensure struct has proper kubebuilder annotations

### Registration Issues

**Error: "chaos type not found in registry"**
- Verify registration in `controllers/types/types.go`
- Check import statements in `controllers/chaosimpl/fx.go`
- Ensure module is added to `AllImpl`

**Error: "action not found"**
- Check action tag in impl.go: `action:"action-name"`
- Verify action constant matches tag value
- Ensure multiplexer is properly configured

### Webhook Validation Failures

**Error: "validation webhook failed"**
- Check webhook validation logic in `*_webhook.go`
- Verify field paths in error messages
- Test validation with various input combinations

**Error: "webhook not registered"**
- Ensure chaos type is in `ChaosObjects`
- Check webhook configuration in Helm chart
- Verify webhook service is running

### Controller Reconciliation Problems

**Error: "controller not reconciling"**
- Check controller logs for errors
- Verify RBAC permissions
- Ensure chaos type is properly registered

**Error: "Apply() not being called"**
- Verify action multiplexer configuration
- Check action tag matches spec.action value
- Review controller logs for routing errors

### Runtime Errors

**Error: "failed to inject chaos"**
- Check target selector matches existing resources
- Verify daemon has necessary permissions
- Review daemon logs for detailed errors

**Error: "chaos not recovering"**
- Implement proper cleanup in Recover()
- Check for resource leaks
- Verify finalizers are handled correctly

## Additional Resources

- [Chaos Mesh Documentation](https://chaos-mesh.org/docs/)
- [Controller Runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Chaos Mesh GitHub Repository](https://github.com/chaos-mesh/chaos-mesh)

## Contributing

When contributing a new chaos type:

1. Follow the implementation guide in this document
2. Ensure all tests pass
3. Update documentation
4. Submit a pull request with clear description
5. Respond to code review feedback

For questions or issues, please open an issue on the Chaos Mesh GitHub repository.
