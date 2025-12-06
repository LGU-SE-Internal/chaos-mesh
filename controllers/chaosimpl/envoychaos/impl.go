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

package envoychaos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/fx"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	impltypes "github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/types"
)

var _ impltypes.ChaosImpl = (*Impl)(nil)

var log logr.Logger

type Impl struct {
	client.Client
	Log logr.Logger
}

// Apply implements the ChaosImpl interface for EnvoyChaos
// EnvoyChaos operates at service level, requires targetService to be specified
func (impl *Impl) Apply(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	impl.Log.Info("envoychaos Apply", "namespace", obj.GetNamespace(), "name", obj.GetName())

	envoychaos := obj.(*v1alpha1.EnvoyChaos)
	
	// EnvoyChaos works at service level, not pod level
	// We only need to process once (on first call)
	if index > 0 {
		return v1alpha1.Injected, nil
	}

	// Validate targetService is specified
	if envoychaos.Spec.TargetService == "" {
		return v1alpha1.NotInjected, fmt.Errorf("targetService must be specified for EnvoyChaos")
	}

	// Determine namespace for the service
	serviceNamespace := envoychaos.Namespace
	if envoychaos.Spec.TargetNamespace != "" {
		serviceNamespace = envoychaos.Spec.TargetNamespace
	} else if envoychaos.Spec.EnvoyConfigNamespace != "" {
		serviceNamespace = envoychaos.Spec.EnvoyConfigNamespace
	}

	// Apply the Envoy configuration for the service
	err := impl.applyEnvoyConfigForService(ctx, envoychaos, serviceNamespace)
	if err != nil {
		impl.Log.Error(err, "failed to apply envoy config", "service", envoychaos.Spec.TargetService)
		return v1alpha1.NotInjected, err
	}

	impl.Log.Info("applied envoy config", "service", envoychaos.Spec.TargetService, "namespace", serviceNamespace)
	return v1alpha1.Injected, nil
}

// Recover implements the ChaosImpl interface for EnvoyChaos
func (impl *Impl) Recover(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	impl.Log.Info("envoychaos Recover", "namespace", obj.GetNamespace(), "name", obj.GetName())

	envoychaos := obj.(*v1alpha1.EnvoyChaos)
	
	// Only need to process once
	if index > 0 {
		return v1alpha1.NotInjected, nil
	}

	// Validate targetService is specified
	if envoychaos.Spec.TargetService == "" {
		return v1alpha1.NotInjected, fmt.Errorf("targetService must be specified for EnvoyChaos")
	}

	// Determine namespace for the service
	serviceNamespace := envoychaos.Namespace
	if envoychaos.Spec.TargetNamespace != "" {
		serviceNamespace = envoychaos.Spec.TargetNamespace
	} else if envoychaos.Spec.EnvoyConfigNamespace != "" {
		serviceNamespace = envoychaos.Spec.EnvoyConfigNamespace
	}

	// Remove the Envoy configuration
	err := impl.removeEnvoyConfig(ctx, envoychaos, serviceNamespace)
	if err != nil {
		if k8sError.IsNotFound(err) {
			return v1alpha1.NotInjected, nil
		}
		impl.Log.Error(err, "failed to remove envoy config", "service", envoychaos.Spec.TargetService)
		return v1alpha1.Injected, err
	}

	impl.Log.Info("removed envoy config", "service", envoychaos.Spec.TargetService, "namespace", serviceNamespace)
	return v1alpha1.NotInjected, nil
}

// applyEnvoyConfigForService creates or updates the CiliumEnvoyConfig for fault injection
func (impl *Impl) applyEnvoyConfigForService(ctx context.Context, envoychaos *v1alpha1.EnvoyChaos, serviceNamespace string) error {
	// Generate the Envoy fault filter configuration
	faultConfig, err := impl.generateFaultConfig(envoychaos)
	if err != nil {
		return err
	}

	targetService := envoychaos.Spec.TargetService
	
	// Create CiliumEnvoyConfig resource name
	configName := fmt.Sprintf("chaos-%s", envoychaos.Name)
	configNamespace := serviceNamespace

	// Create the CiliumEnvoyConfig unstructured object
	config := impl.buildCiliumEnvoyConfig(configName, configNamespace, targetService, serviceNamespace, envoychaos.Name, envoychaos.Spec.TargetPort, faultConfig)

	// Try to create the config
	err = impl.Client.Create(ctx, config)
	if err != nil {
		if k8sError.IsAlreadyExists(err) {
			// Update if it already exists
			err = impl.Client.Update(ctx, config)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	impl.Log.Info("applied envoy config", "name", configName, "namespace", configNamespace, "service", targetService)
	return nil
}

// buildCiliumEnvoyConfig constructs the CiliumEnvoyConfig unstructured object
func (impl *Impl) buildCiliumEnvoyConfig(
	configName, configNamespace string,
	serviceName string,
	serviceNamespace string,
	chaosName string,
	targetPort *int32,
	faultConfig map[string]interface{},
) *unstructured.Unstructured {
	// Build service reference
	serviceRef := map[string]interface{}{
		"name":      serviceName,
		"namespace": serviceNamespace,
	}
	
	// Add port if specified
	if targetPort != nil {
		serviceRef["ports"] = []interface{}{int(*targetPort)}
	}

	config := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumEnvoyConfig",
			"metadata": map[string]interface{}{
				"name":      configName,
				"namespace": configNamespace,
				"labels": map[string]interface{}{
					"chaos-mesh.org/injected": "true",
					"chaos-mesh.org/chaos":    chaosName,
				},
			},
			"spec": map[string]interface{}{
				"services": []interface{}{
					serviceRef,
				},
				"resources": []interface{}{
					map[string]interface{}{
						"@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
						"name":  fmt.Sprintf("chaos-listener-%s", serviceName),
						"filterChains": []interface{}{
							map[string]interface{}{
								"filters": []interface{}{
									map[string]interface{}{
										"name": "envoy.filters.network.http_connection_manager",
										"typedConfig": map[string]interface{}{
											"@type":      "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
											"statPrefix": "chaos_http",
											"httpFilters": []interface{}{
												faultConfig,
												map[string]interface{}{
													"name": "envoy.filters.http.router",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set GVK for the unstructured object
	config.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cilium.io",
		Version: "v2",
		Kind:    "CiliumEnvoyConfig",
	})

	return config
}

// removeEnvoyConfig deletes the CiliumEnvoyConfig for fault injection
func (impl *Impl) removeEnvoyConfig(ctx context.Context, envoychaos *v1alpha1.EnvoyChaos, serviceNamespace string) error {
	configName := fmt.Sprintf("chaos-%s", envoychaos.Name)
	configNamespace := serviceNamespace

	config := &unstructured.Unstructured{}
	config.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cilium.io",
		Version: "v2",
		Kind:    "CiliumEnvoyConfig",
	})

	err := impl.Client.Get(ctx, types.NamespacedName{Name: configName, Namespace: configNamespace}, config)
	if err != nil {
		return err
	}

	err = impl.Client.Delete(ctx, config)
	if err != nil {
		return err
	}

	impl.Log.Info("removed envoy config", "name", configName, "namespace", configNamespace)
	return nil
}

// generateFaultConfig generates the Envoy fault filter configuration based on the chaos spec
func (impl *Impl) generateFaultConfig(envoychaos *v1alpha1.EnvoyChaos) (map[string]interface{}, error) {
	faultConfig := map[string]interface{}{
		"name": "envoy.filters.http.fault",
		"typedConfig": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault",
		},
	}

	typedConfig := faultConfig["typedConfig"].(map[string]interface{})

	// Add delay configuration
	if envoychaos.Spec.Delay != nil && envoychaos.Spec.Delay.FixedDelay != nil {
		delay := map[string]interface{}{
			"fixedDelay": envoychaos.Spec.Delay.FixedDelay,
		}

		percentage := envoychaos.Spec.Percentage
		if envoychaos.Spec.Delay.Percentage != nil {
			percentage = envoychaos.Spec.Delay.Percentage
		}

		if percentage != nil {
			delay["percentage"] = map[string]interface{}{
				"numerator":   int(*percentage),
				"denominator": "HUNDRED",
			}
		}

		typedConfig["delay"] = delay
	}

	// Add abort configuration
	if envoychaos.Spec.Abort != nil {
		abort := map[string]interface{}{}

		// Determine the status code to use
		// For Envoy, grpcStatus can be specified as a string or integer
		// We use integer format for better compatibility
		if envoychaos.Spec.Protocol == "grpc" && envoychaos.Spec.Abort.GrpcStatus != nil {
			abort["grpcStatus"] = int(*envoychaos.Spec.Abort.GrpcStatus)
		} else if envoychaos.Spec.Abort.HTTPStatus != nil {
			abort["httpStatus"] = int(*envoychaos.Spec.Abort.HTTPStatus)
		}

		percentage := envoychaos.Spec.Percentage
		if envoychaos.Spec.Abort.Percentage != nil {
			percentage = envoychaos.Spec.Abort.Percentage
		}

		if percentage != nil {
			abort["percentage"] = map[string]interface{}{
				"numerator":   int(*percentage),
				"denominator": "HUNDRED",
			}
		}

		typedConfig["abort"] = abort
	}

	// Add header matching if specified
	// Headers support exact match by default, or regex if the value starts with "regex:"
	if len(envoychaos.Spec.Headers) > 0 {
		headers := []interface{}{}
		for name, value := range envoychaos.Spec.Headers {
			header := map[string]interface{}{
				"name": name,
			}
			// Support regex matching if value starts with "regex:"
			if strings.HasPrefix(value, "regex:") {
				header["safeRegexMatch"] = map[string]interface{}{
					"regex": strings.TrimPrefix(value, "regex:"),
				}
			} else {
				header["exactMatch"] = value
			}
			headers = append(headers, header)
		}
		typedConfig["headers"] = headers
	}

	// Add upstream cluster if specified
	if envoychaos.Spec.TargetService != "" {
		typedConfig["upstreamCluster"] = envoychaos.Spec.TargetService
	}

	impl.Log.V(1).Info("generated fault config", "config", fmt.Sprintf("%+v", faultConfig))
	return faultConfig, nil
}

// findServiceForPod finds a Kubernetes service that selects the given pod
// parsePodId parses the pod namespace and name from the record id
func parsePodId(id string) (string, string) {
	// Expected format: "namespace/name"
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// NewImpl returns a new EnvoyChaos implementation
func NewImpl(c client.Client, log logr.Logger) *impltypes.ChaosImplPair {
	return &impltypes.ChaosImplPair{
		Name:   "envoychaos",
		Object: &v1alpha1.EnvoyChaos{},
		Impl: &Impl{
			Client: c,
			Log:    log.WithName("envoychaos"),
		},
	}
}

var Module = fx.Provide(
	fx.Annotated{
		Group:  "impl",
		Target: NewImpl,
	},
)

// MarshalJSON is a helper to convert configuration to JSON for debugging
func MarshalJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
