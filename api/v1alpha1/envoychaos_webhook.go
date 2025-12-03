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
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/chaos-mesh/chaos-mesh/api/genericwebhook"
)

// Default sets default values for EnvoyChaosSpec fields
func (in *EnvoyChaosSpec) Default(root interface{}, _ *reflect.StructField) {
	// Set default protocol to grpc if not specified
	if in.Protocol == "" {
		in.Protocol = "grpc"
	}

	// Set default action to fault if not specified
	if in.Action == "" {
		in.Action = EnvoyFaultAction
	}

	// NOTE: Percentage is intentionally not set to a default value
	// Users should explicitly specify the percentage to avoid unintended widespread impact

	// Set default EnvoyConfigNamespace to current namespace if not specified
	if in.EnvoyConfigNamespace == "" && root != nil {
		if chaos, ok := root.(*EnvoyChaos); ok {
			in.EnvoyConfigNamespace = chaos.GetNamespace()
		}
	}
}

// Validate validates EnvoyChaosSpec fields
func (in *EnvoyChaosSpec) Validate(root interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate protocol
	if in.Protocol != "" && in.Protocol != "grpc" && in.Protocol != "http" {
		allErrs = append(allErrs, field.Invalid(path.Child("protocol"), in.Protocol,
			"protocol must be either 'grpc' or 'http'"))
	}

	// Validate action-specific configurations
	switch in.Action {
	case EnvoyDelayAction:
		if in.Delay == nil {
			allErrs = append(allErrs, field.Required(path.Child("delay"),
				"delay configuration is required when action is 'delay'"))
		}

	case EnvoyAbortAction:
		if in.Abort == nil {
			allErrs = append(allErrs, field.Required(path.Child("abort"),
				"abort configuration is required when action is 'abort'"))
		}

	case EnvoyFaultAction:
		// For fault action, at least one of delay or abort should be configured
		if in.Delay == nil && in.Abort == nil {
			allErrs = append(allErrs, field.Invalid(path.Child("action"), in.Action,
				"at least one of delay or abort must be configured when action is 'fault'"))
		}
	}

	// Validate delay configuration if present
	if in.Delay != nil {
		allErrs = append(allErrs, in.Delay.Validate(root, path.Child("delay"))...)
	}

	// Validate abort configuration if present
	if in.Abort != nil {
		allErrs = append(allErrs, in.Abort.Validate(root, path.Child("abort"), in.Protocol)...)
	}

	return allErrs
}

// Validate validates EnvoyDelayConfig fields
func (in *EnvoyDelayConfig) Validate(_ interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if in.FixedDelay == nil {
		allErrs = append(allErrs, field.Required(path.Child("fixedDelay"),
			"fixedDelay is required in delay configuration"))
	} else {
		_, err := time.ParseDuration(*in.FixedDelay)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(path.Child("fixedDelay"), *in.FixedDelay,
				fmt.Sprintf("invalid duration: %s", err)))
		}
	}

	// Validate percentage range
	if in.Percentage != nil && (*in.Percentage < 0 || *in.Percentage > 100) {
		allErrs = append(allErrs, field.Invalid(path.Child("percentage"), *in.Percentage,
			"percentage must be between 0 and 100"))
	}

	return allErrs
}

// Validate validates EnvoyAbortConfig fields
func (in *EnvoyAbortConfig) Validate(_ interface{}, path *field.Path, protocol string) field.ErrorList {
	allErrs := field.ErrorList{}

	// For gRPC, either HTTPStatus or GrpcStatus should be provided
	if protocol == "grpc" && in.GrpcStatus == nil && in.HTTPStatus == nil {
		allErrs = append(allErrs, field.Required(path,
			"either grpcStatus or httpStatus is required for gRPC protocol"))
	}

	// For HTTP, HTTPStatus should be provided
	if protocol == "http" && in.HTTPStatus == nil {
		allErrs = append(allErrs, field.Required(path.Child("httpStatus"),
			"httpStatus is required for HTTP protocol"))
	}

	// Validate percentage range
	if in.Percentage != nil && (*in.Percentage < 0 || *in.Percentage > 100) {
		allErrs = append(allErrs, field.Invalid(path.Child("percentage"), *in.Percentage,
			"percentage must be between 0 and 100"))
	}

	// Validate HTTP status code range
	if in.HTTPStatus != nil && (*in.HTTPStatus < 100 || *in.HTTPStatus > 599) {
		allErrs = append(allErrs, field.Invalid(path.Child("httpStatus"), *in.HTTPStatus,
			"httpStatus must be between 100 and 599"))
	}

	// Validate gRPC status code range (0-16 are valid gRPC status codes)
	if in.GrpcStatus != nil && (*in.GrpcStatus < 0 || *in.GrpcStatus > 16) {
		allErrs = append(allErrs, field.Invalid(path.Child("grpcStatus"), *in.GrpcStatus,
			"grpcStatus must be between 0 and 16"))
	}

	return allErrs
}

func init() {
	genericwebhook.Register("EnvoyChaosSpec", reflect.PtrTo(reflect.TypeOf(EnvoyChaosSpec{})))
}
