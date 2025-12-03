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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var envoychaoslog = logf.Log.WithName("envoychaos-resource")

// +kubebuilder:webhook:path=/mutate-chaos-mesh-org-v1alpha1-envoychaos,mutating=true,failurePolicy=fail,groups=chaos-mesh.org,resources=envoychaos,verbs=create;update,versions=v1alpha1,name=menvoychaos.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.Defaulter = &EnvoyChaos{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *EnvoyChaos) Default() {
	envoychaoslog.Info("default", "name", in.Name)

	in.Spec.Selector.DefaultNamespace(in.GetNamespace())

	// Set default protocol to grpc if not specified
	if in.Spec.Protocol == "" {
		in.Spec.Protocol = "grpc"
	}

	// Set default action to fault if not specified
	if in.Spec.Action == "" {
		in.Spec.Action = EnvoyFaultAction
	}

	// Set default percentage if not specified
	if in.Spec.Percentage == nil {
		defaultPercentage := 100.0
		in.Spec.Percentage = &defaultPercentage
	}

	// Set default EnvoyConfigNamespace to current namespace if not specified
	if in.Spec.EnvoyConfigNamespace == "" {
		in.Spec.EnvoyConfigNamespace = in.GetNamespace()
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-chaos-mesh-org-v1alpha1-envoychaos,mutating=false,failurePolicy=fail,groups=chaos-mesh.org,resources=envoychaos,versions=v1alpha1,name=venvoychaos.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ ChaosValidator = &EnvoyChaos{}
var _ webhook.Validator = &EnvoyChaos{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *EnvoyChaos) ValidateCreate() (admission.Warnings, error) {
	envoychaoslog.Info("validate create", "name", in.Name)
	return in.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *EnvoyChaos) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	envoychaoslog.Info("validate update", "name", in.Name)
	return in.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *EnvoyChaos) ValidateDelete() (admission.Warnings, error) {
	envoychaoslog.Info("validate delete", "name", in.Name)
	return nil, nil
}

// Validate validates chaos object
func (in *EnvoyChaos) Validate() (admission.Warnings, error) {
	specField := field.NewPath("spec")
	allErrs := in.ValidateScheduler(specField)
	allErrs = append(allErrs, in.ValidatePodMode(specField)...)
	allErrs = append(allErrs, in.Spec.validateEnvoyChaosSpec(specField)...)

	if len(allErrs) > 0 {
		return nil, fmt.Errorf(allErrs.ToAggregate().Error())
	}
	return nil, nil
}

// ValidateScheduler validates the scheduler and duration
func (in *EnvoyChaos) ValidateScheduler(spec *field.Path) field.ErrorList {
	return ValidateScheduler(in, spec)
}

// ValidatePodMode validates the value with podmode
func (in *EnvoyChaos) ValidatePodMode(spec *field.Path) field.ErrorList {
	return ValidatePodMode(in.Spec.Value, in.Spec.Mode, spec.Child("value"))
}

func (in *EnvoyChaosSpec) validateEnvoyChaosSpec(spec *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate protocol
	if in.Protocol != "" && in.Protocol != "grpc" && in.Protocol != "http" {
		allErrs = append(allErrs, field.Invalid(spec.Child("protocol"), in.Protocol,
			"protocol must be either 'grpc' or 'http'"))
	}

	// Validate action-specific configurations
	switch in.Action {
	case EnvoyDelayAction:
		if in.Delay == nil {
			allErrs = append(allErrs, field.Required(spec.Child("delay"),
				"delay configuration is required when action is 'delay'"))
		} else if in.Delay.FixedDelay == nil {
			allErrs = append(allErrs, field.Required(spec.Child("delay").Child("fixedDelay"),
				"fixedDelay is required in delay configuration"))
		}

	case EnvoyAbortAction:
		if in.Abort == nil {
			allErrs = append(allErrs, field.Required(spec.Child("abort"),
				"abort configuration is required when action is 'abort'"))
		} else {
			// For gRPC, either HTTPStatus or GrpcStatus should be provided
			if in.Protocol == "grpc" && in.Abort.GrpcStatus == nil && in.Abort.HTTPStatus == nil {
				allErrs = append(allErrs, field.Required(spec.Child("abort"),
					"either grpcStatus or httpStatus is required for gRPC protocol"))
			}
			// For HTTP, HTTPStatus should be provided
			if in.Protocol == "http" && in.Abort.HTTPStatus == nil {
				allErrs = append(allErrs, field.Required(spec.Child("abort").Child("httpStatus"),
					"httpStatus is required for HTTP protocol"))
			}
		}

	case EnvoyFaultAction:
		// For fault action, at least one of delay or abort should be configured
		if in.Delay == nil && in.Abort == nil {
			allErrs = append(allErrs, field.Invalid(spec.Child("action"), in.Action,
				"at least one of delay or abort must be configured when action is 'fault'"))
		}
	}

	// Validate percentage range
	if in.Percentage != nil && (*in.Percentage < 0 || *in.Percentage > 100) {
		allErrs = append(allErrs, field.Invalid(spec.Child("percentage"), *in.Percentage,
			"percentage must be between 0 and 100"))
	}

	// Validate delay percentage if provided
	if in.Delay != nil && in.Delay.Percentage != nil {
		if *in.Delay.Percentage < 0 || *in.Delay.Percentage > 100 {
			allErrs = append(allErrs, field.Invalid(spec.Child("delay").Child("percentage"),
				*in.Delay.Percentage, "delay percentage must be between 0 and 100"))
		}
	}

	// Validate abort percentage if provided
	if in.Abort != nil && in.Abort.Percentage != nil {
		if *in.Abort.Percentage < 0 || *in.Abort.Percentage > 100 {
			allErrs = append(allErrs, field.Invalid(spec.Child("abort").Child("percentage"),
				*in.Abort.Percentage, "abort percentage must be between 0 and 100"))
		}
	}

	// Validate HTTP status code range
	if in.Abort != nil && in.Abort.HTTPStatus != nil {
		if *in.Abort.HTTPStatus < 100 || *in.Abort.HTTPStatus > 599 {
			allErrs = append(allErrs, field.Invalid(spec.Child("abort").Child("httpStatus"),
				*in.Abort.HTTPStatus, "httpStatus must be between 100 and 599"))
		}
	}

	// Validate gRPC status code range (0-16 are valid gRPC status codes)
	if in.Abort != nil && in.Abort.GrpcStatus != nil {
		if *in.Abort.GrpcStatus < 0 || *in.Abort.GrpcStatus > 16 {
			allErrs = append(allErrs, field.Invalid(spec.Child("abort").Child("grpcStatus"),
				*in.Abort.GrpcStatus, "grpcStatus must be between 0 and 16"))
		}
	}

	return allErrs
}
