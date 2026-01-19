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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("envoychaos_types", func() {
	Context("GetSelectorSpecs", func() {
		It("should return selector specs", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-envoychaos",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					PodSelector: PodSelector{
						Mode: AllMode,
					},
				},
			}

			specs := envoychaos.GetSelectorSpecs()
			Expect(specs).To(HaveLen(1))
			Expect(specs).To(HaveKey("."))
		})
	})

	Context("GetCustomStatus", func() {
		It("should return instances status", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-envoychaos",
					Namespace: metav1.NamespaceDefault,
				},
				Status: EnvoyChaosStatus{
					Instances: map[string]int64{
						"pod-1": 1,
						"pod-2": 2,
					},
				},
			}

			status := envoychaos.GetCustomStatus()
			instancesPtr, ok := status.(*map[string]int64)
			Expect(ok).To(BeTrue())
			Expect(*instancesPtr).To(HaveLen(2))
			Expect(*instancesPtr).To(HaveKey("pod-1"))
			Expect(*instancesPtr).To(HaveKey("pod-2"))
		})
	})
})
