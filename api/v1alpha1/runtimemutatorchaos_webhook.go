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

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const DefaultRuntimeMutatorPort int32 = 9090

func (in *RuntimeMutatorChaosSpec) Default(root interface{}, field *reflect.StructField) {
	if in == nil {
		return
	}

	if in.Port == 0 {
		in.Port = DefaultRuntimeMutatorPort
	}
}

func (in *RuntimeMutatorChaosSpec) Validate(root interface{}, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	switch in.Action {
	case RuntimeMutatorConstantAction:
		if in.From == nil || len(*in.From) == 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "from field must be provided for constant mutation"))
		}
		if in.To == nil || len(*in.To) == 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "to field must be provided for constant mutation"))
		}
		if in.Strategy != nil && len(*in.Strategy) != 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "strategy field should not be set for constant mutation"))
		}

	case RuntimeMutatorOperatorAction, RuntimeMutatorStringAction:
		if in.Strategy == nil || len(*in.Strategy) == 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "strategy field must be provided for operator/string mutation"))
		}
		if in.From != nil && len(*in.From) != 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "from field should not be set for operator/string mutation"))
		}
		if in.To != nil && len(*in.To) != 0 {
			allErrs = append(allErrs, field.Invalid(path, in, "to field should not be set for operator/string mutation"))
		}

	case "":
		allErrs = append(allErrs, field.Invalid(path, in, "action not provided"))
	default:
		allErrs = append(allErrs, field.Invalid(path, in, fmt.Sprintf("action %s not supported, action can be 'constant', 'operator' or 'string'", in.Action)))
	}

	// Common validations for all actions
	if len(in.Class) == 0 {
		allErrs = append(allErrs, field.Invalid(path, in, "class not provided"))
	}

	if len(in.Method) == 0 {
		allErrs = append(allErrs, field.Invalid(path, in, "method not provided"))
	}

	return allErrs
}