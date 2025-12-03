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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="duration",type=string,JSONPath=`.spec.duration`
// +chaos-mesh:experiment
// +genclient

// EnvoyChaos is the Schema for the envoychaos API
type EnvoyChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyChaosSpec   `json:"spec,omitempty"`
	Status EnvoyChaosStatus `json:"status,omitempty"`
}

var _ InnerObjectWithCustomStatus = (*EnvoyChaos)(nil)
var _ InnerObjectWithSelector = (*EnvoyChaos)(nil)
var _ InnerObject = (*EnvoyChaos)(nil)

// EnvoyChaosAction represents the chaos action about Envoy proxy.
type EnvoyChaosAction string

const (
	// EnvoyFaultAction represents the chaos action of injecting faults via Envoy fault filter.
	EnvoyFaultAction EnvoyChaosAction = "fault"
	
	// EnvoyDelayAction represents the chaos action of adding delay to gRPC/HTTP requests.
	EnvoyDelayAction EnvoyChaosAction = "delay"
	
	// EnvoyAbortAction represents the chaos action of aborting gRPC/HTTP requests.
	EnvoyAbortAction EnvoyChaosAction = "abort"
)

// EnvoyChaosSpec defines the desired state of EnvoyChaos
type EnvoyChaosSpec struct {
	PodSelector `json:",inline"`

	// Action defines the specific Envoy chaos action.
	// Supported action: fault, delay, abort
	// Default action: fault
	// +kubebuilder:validation:Enum=fault;delay;abort
	Action EnvoyChaosAction `json:"action"`

	// EnvoyConfigName is the name of the CiliumEnvoyConfig or EnvoyFilter to inject faults.
	// If not provided, will auto-detect the Envoy configuration.
	// +optional
	EnvoyConfigName string `json:"envoyConfigName,omitempty"`

	// EnvoyConfigNamespace is the namespace of the CiliumEnvoyConfig or EnvoyFilter.
	// Defaults to the same namespace as the chaos object.
	// +optional
	EnvoyConfigNamespace string `json:"envoyConfigNamespace,omitempty"`

	// Protocol defines the protocol type for fault injection.
	// +kubebuilder:validation:Enum=grpc;http
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Delay represents the delay configuration for fault injection.
	// +optional
	Delay *EnvoyDelayConfig `json:"delay,omitempty"`

	// Abort represents the abort configuration for fault injection.
	// +optional
	Abort *EnvoyAbortConfig `json:"abort,omitempty"`

	// Percentage is the percentage of requests to which the fault will be injected.
	// Valid range is 0 to 100 (whole numbers only, e.g., 50 = 50%)
	// If not specified, fault injection applies to all matching requests
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage *int32 `json:"percentage,omitempty"`

	// Path is a rule to select target by uri path in http/grpc request.
	// Support exact match, prefix match and regex match.
	// +optional
	Path *string `json:"path,omitempty"`

	// Method is a rule to select target by grpc method or http method in request.
	// +optional
	Method *string `json:"method,omitempty"`

	// Headers is a rule to select target by headers in request.
	// The key-value pairs represent header name and header value pairs.
	// Values support exact matching by default. For regex matching, prefix the value with "regex:".
	// Example: {"x-user-id": "123"} for exact match, {"x-user-id": "regex:^test-.*"} for regex.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Duration represents the duration of the chaos action.
	// +optional
	Duration *string `json:"duration,omitempty" webhook:"Duration"`

	// RemoteCluster represents the remote cluster where the chaos will be deployed
	// +optional
	RemoteCluster string `json:"remoteCluster,omitempty"`

	// TargetService specifies the Kubernetes service to target for fault injection.
	// +optional
	TargetService string `json:"targetService,omitempty"`

	// TargetPort specifies the port number of the target service.
	// +optional
	TargetPort *int32 `json:"targetPort,omitempty"`
}

// EnvoyDelayConfig defines the delay configuration for Envoy fault injection
type EnvoyDelayConfig struct {
	// FixedDelay represents the fixed delay duration.
	// A duration string is a possibly unsigned sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// +optional
	FixedDelay *string `json:"fixedDelay,omitempty" webhook:"Delay"`

	// Percentage is the percentage of requests to which delay will be injected.
	// Valid range is 0 to 100 (whole numbers only). If not specified, inherits from parent.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage *int32 `json:"percentage,omitempty"`
}

// EnvoyAbortConfig defines the abort configuration for Envoy fault injection
type EnvoyAbortConfig struct {
	// HTTPStatus represents the HTTP status code to return when aborting.
	// For gRPC, this will be mapped to appropriate gRPC status code.
	// +optional
	HTTPStatus *int32 `json:"httpStatus,omitempty"`

	// GrpcStatus represents the gRPC status code to return when aborting gRPC requests.
	// If not specified for gRPC, defaults to UNAVAILABLE (14).
	// +optional
	GrpcStatus *int32 `json:"grpcStatus,omitempty"`

	// Percentage is the percentage of requests to abort.
	// Valid range is 0 to 100 (whole numbers only). If not specified, inherits from parent.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage *int32 `json:"percentage,omitempty"`
}

type EnvoyChaosStatus struct {
	ChaosStatus `json:",inline"`

	// Instances always specifies envoy chaos generation or empty
	// +optional
	Instances map[string]int64 `json:"instances,omitempty"`
}

func (obj *EnvoyChaos) GetSelectorSpecs() map[string]interface{} {
	return map[string]interface{}{
		".": &obj.Spec.PodSelector,
	}
}

func (obj *EnvoyChaos) GetCustomStatus() interface{} {
	return &obj.Status.Instances
}
