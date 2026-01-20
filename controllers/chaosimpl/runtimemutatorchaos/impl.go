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
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/types"
	"github.com/chaos-mesh/chaos-mesh/controllers/chaosimpl/utils"
)

type Impl struct {
	client.Client
	Log     logr.Logger
	decoder *utils.ContainerRecordDecoder
}

// Apply applies RuntimeMutatorChaos
func (impl *Impl) Apply(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	impl.Log.Info("Apply RuntimeMutatorChaos")

	if impl.decoder == nil {
		return v1alpha1.NotInjected, errors.New("decoder is nil")
	}

	runtimeMutatorChaos := obj.(*v1alpha1.RuntimeMutatorChaos)

	decodedContainer, err := impl.decoder.DecodeContainerRecord(ctx, records[index], obj)
	if decodedContainer.PbClient != nil {
		defer func() {
			err := decodedContainer.PbClient.Close()
			if err != nil {
				impl.Log.Error(err, "fail to close pb client")
			}
		}()
	}
	if err != nil {
		return v1alpha1.NotInjected, err
	}

	// Build the mutation configuration
	// TODO: Implement protobuf message in cm-004
	impl.Log.Info("installing runtime mutator", "container", decodedContainer.ContainerId, "action", runtimeMutatorChaos.Spec.Action, "class", runtimeMutatorChaos.Spec.Class, "method", runtimeMutatorChaos.Spec.Method)

	// Validate the spec based on action type
	switch runtimeMutatorChaos.Spec.Action {
	case v1alpha1.RuntimeMutatorConstantAction:
		if runtimeMutatorChaos.Spec.From == nil || runtimeMutatorChaos.Spec.To == nil {
			return v1alpha1.NotInjected, errors.New("from and to fields are required for constant mutation")
		}
	case v1alpha1.RuntimeMutatorOperatorAction, v1alpha1.RuntimeMutatorStringAction:
		if runtimeMutatorChaos.Spec.Strategy == nil {
			return v1alpha1.NotInjected, errors.New("strategy field is required for operator/string mutation")
		}
	}

	// TODO: Call decodedContainer.PbClient.InstallRuntimeMutator when protobuf is implemented
	// For now, just log that we would install the mutator
	impl.Log.Info("would install runtime mutator", "container", decodedContainer.ContainerId, "action", runtimeMutatorChaos.Spec.Action)

	return v1alpha1.Injected, nil
}

// Recover recovers the RuntimeMutatorChaos
func (impl *Impl) Recover(ctx context.Context, index int, records []*v1alpha1.Record, obj v1alpha1.InnerObject) (v1alpha1.Phase, error) {
	impl.Log.Info("Recover RuntimeMutatorChaos")

	if impl.decoder == nil {
		return v1alpha1.Injected, errors.New("decoder is nil")
	}

	decodedContainer, err := impl.decoder.DecodeContainerRecord(ctx, records[index], obj)
	if decodedContainer.PbClient != nil {
		defer func() {
			err := decodedContainer.PbClient.Close()
			if err != nil {
				impl.Log.Error(err, "fail to close pb client")
			}
		}()
	}
	if err != nil {
		return v1alpha1.Injected, err
	}

	impl.Log.Info("uninstalling runtime mutator", "container", decodedContainer.ContainerId)

	// TODO: Call decodedContainer.PbClient.UninstallRuntimeMutator when protobuf is implemented
	// For now, just log that we would uninstall the mutator
	impl.Log.Info("would uninstall runtime mutator", "container", decodedContainer.ContainerId)

	return v1alpha1.NotInjected, nil
}

// NewImpl creates a new RuntimeMutatorChaos implementation
func NewImpl(client client.Client, log logr.Logger, decoder *utils.ContainerRecordDecoder) *types.ChaosImplPair {
	return &types.ChaosImplPair{
		Name:   "runtimemutatorchaos",
		Object: &v1alpha1.RuntimeMutatorChaos{},
		Impl: &Impl{
			Client:  client,
			Log:     log.WithName("runtimemutatorchaos"),
			decoder: decoder,
		},
		ObjectList: &v1alpha1.RuntimeMutatorChaosList{},
	}
}

// Module creates a new fx.Module for RuntimeMutatorChaos
var Module = fx.Provide(
	func(client client.Client, log logr.Logger, decoder *utils.ContainerRecordDecoder) *types.ChaosImplPair {
		return NewImpl(client, log, decoder)
	},
)
