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

var _ = Describe("envoychaos_webhook", func() {
	Context("Defaulter", func() {
		It("set default namespace selector", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault},
			}
			envoychaos.Default()
			Expect(envoychaos.Spec.Selector.Namespaces[0]).To(Equal(metav1.NamespaceDefault))
		})

		It("set default protocol to grpc", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault},
			}
			envoychaos.Default()
			Expect(envoychaos.Spec.Protocol).To(Equal("grpc"))
		})

		It("set default action to fault", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault},
			}
			envoychaos.Default()
			Expect(envoychaos.Spec.Action).To(Equal(EnvoyFaultAction))
		})

		It("set default percentage to 100", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault},
			}
			envoychaos.Default()
			Expect(*envoychaos.Spec.Percentage).To(Equal(100.0))
		})

		It("set default envoyConfigNamespace to current namespace", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace"},
			}
			envoychaos.Default()
			Expect(envoychaos.Spec.EnvoyConfigNamespace).To(Equal("test-namespace"))
		})
	})

	Context("Validator", func() {
		It("Validate delay action", func() {
			fixedDelay := "100ms"
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delay",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action: EnvoyDelayAction,
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Delay: &EnvoyDelayConfig{
						FixedDelay: &fixedDelay,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).To(BeNil())
		})

		It("Validate abort action with gRPC status", func() {
			grpcStatus := int32(14) // UNAVAILABLE
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-abort-grpc",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:   EnvoyAbortAction,
					Protocol: "grpc",
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Abort: &EnvoyAbortConfig{
						GrpcStatus: &grpcStatus,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).To(BeNil())
		})

		It("Validate abort action with HTTP status", func() {
			httpStatus := int32(500)
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-abort-http",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:   EnvoyAbortAction,
					Protocol: "http",
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Abort: &EnvoyAbortConfig{
						HTTPStatus: &httpStatus,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).To(BeNil())
		})

		It("should reject delay action without delay config", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-delay",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action: EnvoyDelayAction,
					PodSelector: PodSelector{
						Mode: AllMode,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})

		It("should reject abort action without abort config for grpc", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-abort",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:   EnvoyAbortAction,
					Protocol: "grpc",
					PodSelector: PodSelector{
						Mode: AllMode,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})

		It("should reject fault action without delay or abort", func() {
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-fault",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action: EnvoyFaultAction,
					PodSelector: PodSelector{
						Mode: AllMode,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})

		It("should reject invalid percentage", func() {
			fixedDelay := "100ms"
			invalidPercentage := 150.0
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-percentage",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:     EnvoyDelayAction,
					Percentage: &invalidPercentage,
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Delay: &EnvoyDelayConfig{
						FixedDelay: &fixedDelay,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})

		It("should reject invalid HTTP status code", func() {
			invalidHTTPStatus := int32(1000)
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-http-status",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:   EnvoyAbortAction,
					Protocol: "http",
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Abort: &EnvoyAbortConfig{
						HTTPStatus: &invalidHTTPStatus,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})

		It("should reject invalid gRPC status code", func() {
			invalidGrpcStatus := int32(20)
			envoychaos := &EnvoyChaos{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-grpc-status",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: EnvoyChaosSpec{
					Action:   EnvoyAbortAction,
					Protocol: "grpc",
					PodSelector: PodSelector{
						Mode: AllMode,
					},
					Abort: &EnvoyAbortConfig{
						GrpcStatus: &invalidGrpcStatus,
					},
				},
			}

			_, err := envoychaos.ValidateCreate()
			Expect(err).ToNot(BeNil())
		})
	})
})
