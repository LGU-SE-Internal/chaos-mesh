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

// RuntimeMutatorChaosSpec defines the desired state of RuntimeMutatorChaos
type RuntimeMutatorChaosSpec struct {
	ContainerSelector `json:",inline"`

	// Duration represents the duration of the chaos action
	// +optional
	Duration *string `json:"duration,omitempty" webhook:"Duration"`

	// Action defines the specific runtime mutator chaos action.
	// Supported action: constant;operator;string
	// +kubebuilder:validation:Enum=constant;operator;string
	Action RuntimeMutatorChaosAction `json:"action"`

	// RuntimeMutatorParameter represents the detail about runtime mutator chaos action definition
	// +optional
	RuntimeMutatorParameter `json:",inline"`

	// RemoteCluster represents the remote cluster where the chaos will be deployed
	// +optional
	RemoteCluster string `json:"remoteCluster,omitempty"`
}

// RuntimeMutatorChaosAction represents the chaos action about runtime mutator
type RuntimeMutatorChaosAction string

const (
	// RuntimeMutatorConstantAction represents the runtime mutator chaos action of constant value mutation
	RuntimeMutatorConstantAction RuntimeMutatorChaosAction = "constant"

	// RuntimeMutatorOperatorAction represents the runtime mutator chaos action of operator mutation
	RuntimeMutatorOperatorAction RuntimeMutatorChaosAction = "operator"

	// RuntimeMutatorStringAction represents the runtime mutator chaos action of string mutation
	RuntimeMutatorStringAction RuntimeMutatorChaosAction = "string"
)

// RuntimeMutatorParameter represents the detail about runtime mutator chaos action definition
type RuntimeMutatorParameter struct {
	// Java class to target for mutation
	// +optional
	Class string `json:"class,omitempty"`

	// Method in the Java class to target for mutation
	// +optional
	Method string `json:"method,omitempty"`

	// For constant mutation: the original value to replace
	// +optional
	From string `json:"from,omitempty"`

	// For constant mutation: the new value to inject
	// +optional
	To string `json:"to,omitempty"`

	// For operator/string mutation: the mutation strategy
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// The port of the runtime mutator agent server, default 9090
	// +optional
	Port int32 `json:"port,omitempty"`
}

// RuntimeMutatorChaosStatus defines the observed state of RuntimeMutatorChaos
type RuntimeMutatorChaosStatus struct {
	ChaosStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="duration",type=string,JSONPath=`.spec.duration`
// +chaos-mesh:experiment
// +genclient

// RuntimeMutatorChaos is the Schema for the runtimemutatorchaos API
type RuntimeMutatorChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuntimeMutatorChaosSpec   `json:"spec,omitempty"`
	Status RuntimeMutatorChaosStatus `json:"status,omitempty"`
}

var _ InnerObjectWithSelector = (*RuntimeMutatorChaos)(nil)
var _ InnerObject = (*RuntimeMutatorChaos)(nil)

func init() {
	SchemeBuilder.Register(&RuntimeMutatorChaos{}, &RuntimeMutatorChaosList{})
}

func (obj *RuntimeMutatorChaos) GetSelectorSpecs() map[string]interface{} {
	return map[string]interface{}{
		".": &obj.Spec.ContainerSelector,
	}
}