/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/NVIDIA/topograph/internal/httperr"
	"github.com/NVIDIA/topograph/internal/models"
	"github.com/NVIDIA/topograph/pkg/providers"
	"github.com/NVIDIA/topograph/pkg/topology"
)

const NAME_SIM = "crusoe-sim"

type ProviderSim struct {
	model *models.Model
}

func NamedLoaderSim() (string, providers.Loader) {
	return NAME_SIM, LoaderSim
}

// LoaderSim creates simulation provider from YAML model
func LoaderSim(ctx context.Context, config providers.Config) (providers.Provider, *httperr.Error) {
	model, err := models.LoadModel(config.ModelFile)
	if err != nil {
		return nil, httperr.NewError(http.StatusBadRequest, fmt.Sprintf("failed to load model: %v", err))
	}

	klog.Infof("Created Crusoe simulation provider from model: %s", config.ModelFile)
	return NewSim(model), nil
}

func NewSim(model *models.Model) *ProviderSim {
	return &ProviderSim{
		model: model,
	}
}

// GenerateTopologyConfig builds topology from YAML model
func (p *ProviderSim) GenerateTopologyConfig(ctx context.Context, pageSize *int, instances []topology.ComputeInstances) (*topology.Vertex, *httperr.Error) {
	// Validate multi-region constraint
	if len(instances) > 1 {
		return nil, httperr.NewError(http.StatusBadRequest, "Crusoe does not support multi-region topology requests")
	}

	// Build instance topology from model
	topo, err := p.buildTopologyFromModel(instances)
	if err != nil {
		return nil, err
	}

	klog.Infof("Built simulation topology with %d instances", topo.Len())

	// Convert to 3-tier graph
	return topo.ToThreeTierGraph(NAME, instances, false), nil
}

// buildTopologyFromModel extracts topology from YAML model
func (p *ProviderSim) buildTopologyFromModel(instances []topology.ComputeInstances) (*topology.ClusterTopology, *httperr.Error) {
	// Build instance filter if specified
	var requestedIDs map[string]struct{}
	if len(instances) > 0 && len(instances[0].Instances) > 0 {
		requestedIDs = make(map[string]struct{}, len(instances[0].Instances))
		for id := range instances[0].Instances {
			requestedIDs[id] = struct{}{}
		}
	}

	topo := topology.NewClusterTopology()
	var stats extractionStats

	// Iterate through capacity blocks to find compute instances
	for _, cb := range p.model.CapacityBlocks {
		// Find the pod this capacity block belongs to
		podSwitch := p.findPodForCapacityBlock(cb.Name)
		if podSwitch == nil {
			klog.V(4).Infof("Capacity block %q has no parent pod switch, skipping", cb.Name)
			continue
		}

		// Extract topology labels from switch metadata
		partitionID, switchID, err := extractSimTopologyLabels(podSwitch.Metadata)
		if err != nil {
			klog.V(4).Infof("Switch %q missing topology metadata: %v", podSwitch.Name, err)
			stats.badLabels++
			continue
		}

		// Process nodes in this capacity block
		for _, nodeName := range cb.Nodes {
			// Apply filter if specified
			if requestedIDs != nil {
				if _, exists := requestedIDs[nodeName]; !exists {
					stats.skipped++
					continue
				}
			}

			instance := &topology.InstanceTopology{
				InstanceID:     nodeName,
				DatacenterID:   partitionID,
				SpineID:        switchID,
				BlockID:        switchID,
				DatacenterName: partitionID,
				SpineName:      "switch-" + switchID,
				BlockName:      "switch-" + switchID,
			}

			topo.Append(instance)
			stats.success++
		}
	}

	// Record metrics
	nodesProcessed.WithLabelValues("success").Add(float64(stats.success))
	nodesProcessed.WithLabelValues("skipped").Add(float64(stats.skipped))
	nodesProcessed.WithLabelValues("missing_labels").Add(float64(stats.badLabels))

	klog.Infof("Simulation topology: %d added, %d skipped, %d missing labels", stats.success, stats.skipped, stats.badLabels)

	// Handle empty result
	if stats.success == 0 {
		return nil, resolveEmptyTopologyError(stats)
	}

	return topo, nil
}

// findPodForCapacityBlock finds the pod switch that contains this capacity block
func (p *ProviderSim) findPodForCapacityBlock(cbName string) *models.Switch {
	for _, sw := range p.model.Switches {
		// Check if this switch has capacity blocks (it's a pod switch)
		for _, cb := range sw.CapacityBlocks {
			if cb == cbName {
				return &sw
			}
		}
	}
	return nil
}

// extractSimTopologyLabels extracts partition_id and switch_id from switch metadata
func extractSimTopologyLabels(metadata map[string]string) (partitionID, switchID string, err error) {
	partitionID, ok := metadata["partition_id"]
	if !ok || len(partitionID) == 0 {
		return "", "", fmt.Errorf("missing or empty metadata key 'partition_id'")
	}

	switchID, ok = metadata["switch_id"]
	if !ok || len(switchID) == 0 {
		return "", "", fmt.Errorf("missing or empty metadata key 'switch_id'")
	}

	return partitionID, switchID, nil
}

// GetComputeInstances extracts all compute instances from the model
func (p *ProviderSim) GetComputeInstances(ctx context.Context) ([]topology.ComputeInstances, *httperr.Error) {
	instances := make(map[string]struct{})

	for _, cb := range p.model.CapacityBlocks {
		for _, node := range cb.Nodes {
			instances[node] = struct{}{}
		}
	}

	// Convert to ComputeInstances format
	result := topology.ComputeInstances{
		Region:    "local",
		Instances: instances,
	}

	return []topology.ComputeInstances{result}, nil
}
