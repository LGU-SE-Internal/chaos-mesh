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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestRuntimeMutatorChaosSpecValidation(t *testing.T) {
	testCases := []struct {
		name        string
		spec        RuntimeMutatorChaosSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid constant mutation",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
					From:   "100",
					To:     "0",
				},
			},
			expectError: false,
		},
		{
			name: "valid operator mutation",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorOperatorAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:    "TestClass",
					Method:   "testMethod",
					Strategy: "add-to-sub",
				},
			},
			expectError: false,
		},
		{
			name: "valid string mutation",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorStringAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:    "TestClass",
					Method:   "testMethod",
					Strategy: "return-empty",
				},
			},
			expectError: false,
		},
		{
			name: "missing action",
			spec: RuntimeMutatorChaosSpec{
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
				},
			},
			expectError: true,
			errorMsg:    "action not provided",
		},
		{
			name: "constant mutation missing from field",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
					To:     "0",
				},
			},
			expectError: true,
			errorMsg:    "from field must be provided for constant mutation",
		},
		{
			name: "constant mutation missing to field",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
					From:   "100",
				},
			},
			expectError: true,
			errorMsg:    "to field must be provided for constant mutation",
		},
		{
			name: "constant mutation with strategy field",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:    "TestClass",
					Method:   "testMethod",
					From:     "100",
					To:       "0",
					Strategy: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "strategy field should not be set for constant mutation",
		},
		{
			name: "operator mutation missing strategy",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorOperatorAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
				},
			},
			expectError: true,
			errorMsg:    "strategy field must be provided for operator/string mutation",
		},
		{
			name: "operator mutation with from field",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorOperatorAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:    "TestClass",
					Method:   "testMethod",
					Strategy: "add-to-sub",
					From:     "invalid",
				},
			},
			expectError: true,
			errorMsg:    "from field should not be set for operator/string mutation",
		},
		{
			name: "string mutation with to field",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorStringAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:    "TestClass",
					Method:   "testMethod",
					Strategy: "return-empty",
					To:       "invalid",
				},
			},
			expectError: true,
			errorMsg:    "to field should not be set for operator/string mutation",
		},
		{
			name: "missing class",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Method: "testMethod",
					From:   "100",
					To:     "0",
				},
			},
			expectError: true,
			errorMsg:    "class not provided",
		},
		{
			name: "missing method",
			spec: RuntimeMutatorChaosSpec{
				Action: RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					From:   "100",
					To:     "0",
				},
			},
			expectError: true,
			errorMsg:    "method not provided",
		},
		{
			name: "invalid action",
			spec: RuntimeMutatorChaosSpec{
				Action: "invalid-action",
				RuntimeMutatorParameter: RuntimeMutatorParameter{
					Class:  "TestClass",
					Method: "testMethod",
				},
			},
			expectError: true,
			errorMsg:    "action invalid-action not supported",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Validate(nil, field.NewPath("spec"))
			if tc.expectError {
				assert.NotEmpty(t, err)
				if tc.errorMsg != "" {
					found := false
					for _, e := range err {
						if e.Detail == tc.errorMsg || (e.Detail != "" && len(e.Detail) > 0 && len(tc.errorMsg) > 0 && e.Detail[0:min(len(e.Detail), len(tc.errorMsg))] == tc.errorMsg[0:min(len(e.Detail), len(tc.errorMsg))]) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message containing '%s'", tc.errorMsg)
				}
			} else {
				assert.Empty(t, err)
			}
		})
	}
}

func TestRuntimeMutatorChaosSpecDefault(t *testing.T) {
	spec := &RuntimeMutatorChaosSpec{
		Action: RuntimeMutatorConstantAction,
		RuntimeMutatorParameter: RuntimeMutatorParameter{
			Class:  "TestClass",
			Method: "testMethod",
		},
	}

	spec.Default(nil, nil)

	assert.Equal(t, DefaultRuntimeMutatorPort, spec.Port)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}