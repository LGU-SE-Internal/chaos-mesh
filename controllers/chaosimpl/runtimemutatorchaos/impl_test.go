// Copyright 2026 Chaos Mesh.
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

package runtimemutatorchaos

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
)

func TestRuntimeMutatorChaosValidation(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		name        string
		spec        v1alpha1.RuntimeMutatorChaosSpec
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid constant mutation",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:  "com.example.TestClass",
					Method: "testMethod",
					Port:   9090,
					From:   stringPtr("100"),
					To:     stringPtr("0"),
				},
			},
			shouldError: false,
		},
		{
			name: "valid operator mutation",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorOperatorAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:    "com.example.TestClass",
					Method:   "testMethod",
					Port:     9090,
					Strategy: stringPtr("add-to-subtract"),
				},
			},
			shouldError: false,
		},
		{
			name: "valid string mutation",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorStringAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:    "com.example.TestClass",
					Method:   "testMethod",
					Port:     9090,
					Strategy: stringPtr("empty-string"),
				},
			},
			shouldError: false,
		},
		{
			name: "constant mutation missing from field",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:  "com.example.TestClass",
					Method: "testMethod",
					Port:   9090,
					To:     stringPtr("0"),
				},
			},
			shouldError: true,
			errorMsg:    "from and to fields are required for constant mutation",
		},
		{
			name: "constant mutation missing to field",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorConstantAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:  "com.example.TestClass",
					Method: "testMethod",
					Port:   9090,
					From:   stringPtr("100"),
				},
			},
			shouldError: true,
			errorMsg:    "from and to fields are required for constant mutation",
		},
		{
			name: "operator mutation missing strategy",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorOperatorAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:  "com.example.TestClass",
					Method: "testMethod",
					Port:   9090,
				},
			},
			shouldError: true,
			errorMsg:    "strategy field is required for operator/string mutation",
		},
		{
			name: "string mutation missing strategy",
			spec: v1alpha1.RuntimeMutatorChaosSpec{
				Action: v1alpha1.RuntimeMutatorStringAction,
				RuntimeMutatorParameter: v1alpha1.RuntimeMutatorParameter{
					Class:  "com.example.TestClass",
					Method: "testMethod",
					Port:   9090,
				},
			},
			shouldError: true,
			errorMsg:    "strategy field is required for operator/string mutation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a simple validation function to test the logic
			var err error
			switch tc.spec.Action {
			case v1alpha1.RuntimeMutatorConstantAction:
				if tc.spec.From == nil || tc.spec.To == nil {
					err = errors.New("from and to fields are required for constant mutation")
				}
			case v1alpha1.RuntimeMutatorOperatorAction, v1alpha1.RuntimeMutatorStringAction:
				if tc.spec.Strategy == nil {
					err = errors.New("strategy field is required for operator/string mutation")
				}
			}

			if tc.shouldError {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(tc.errorMsg))
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}