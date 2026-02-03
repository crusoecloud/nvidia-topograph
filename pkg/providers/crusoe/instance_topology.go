/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/NVIDIA/topograph/internal/httperr"
	"github.com/NVIDIA/topograph/internal/k8s"
	"github.com/NVIDIA/topograph/pkg/topology"
)

type extractionStats struct {
	success   int
	skipped   int
	badLabels int
}

// generateInstanceTopology builds ClusterTopology from K8s node labels
func (p *baseProvider) generateInstanceTopology(ctx context.Context, instances []topology.ComputeInstances) (*topology.ClusterTopology, *httperr.Error) {
	// Prepare the filter set
	requestedIDs, err := getRequestedNodeIDs(instances)
	if err != nil {
		return nil, err
	}

	// Fetch all nodes
	nodeList, k8sErr := k8s.GetNodes(ctx, p.kubeClient, p.params.nodeListOpt)

	if k8sErr != nil {
		return nil, httperr.NewError(http.StatusBadGateway, fmt.Sprintf("failed to list nodes: %v", k8sErr))
	}

	topo := topology.NewClusterTopology()
	var stats extractionStats

	klog.Infof("Processing %d nodes (filtering: %v)", len(nodeList.Items), len(requestedIDs) > 0)

	for _, node := range nodeList.Items {
		if shouldSkipNode(node.Name, requestedIDs) {
			stats.skipped++
			continue
		}

		instance := buildInstanceTopology(node)
		klog.V(4).Infof("Built instance topology: ID=%q datacenter=%q spine=%q block=%q", 
			instance.InstanceID, instance.DatacenterID, instance.SpineID, instance.BlockID)
		topo.Append(instance)
		stats.success++
	}

	klog.Infof("Topology extraction: %d added, %d skipped, %d missing labels", stats.success, stats.skipped, stats.badLabels)

	// Record metrics
	nodesProcessed.WithLabelValues("success").Add(float64(stats.success))
	nodesProcessed.WithLabelValues("skipped").Add(float64(stats.skipped))
	nodesProcessed.WithLabelValues("missing_labels").Add(float64(stats.badLabels))

	// Handle empty result scenarios
	if stats.success == 0 {
		return nil, resolveEmptyTopologyError(stats)
	}

	return topo, nil
}

// getRequestedNodeIDs validates input and returns a lookup set of IDs
func getRequestedNodeIDs(instances []topology.ComputeInstances) (map[string]struct{}, *httperr.Error) {
	if len(instances) > 1 {
		return nil, httperr.NewError(http.StatusBadRequest, "Crusoe does not support multi-region topology requests")
	}

	if len(instances) == 0 || len(instances[0].Instances) == 0 {
		return nil, nil
	}

	ids := make(map[string]struct{}, len(instances[0].Instances))
	for id := range instances[0].Instances {
		ids[id] = struct{}{}
	}
	return ids, nil
}

// shouldSkipNode checks if the node should be filtered out
func shouldSkipNode(nodeName string, requestedIDs map[string]struct{}) bool {
	if requestedIDs == nil {
		return false
	}
	_, exists := requestedIDs[nodeName]
	return !exists
}

// buildInstanceTopology constructs topology object from node labels.
// Uses 3-tier hierarchy with common root: Crusoe (L1) -> Partition (L2) -> Pod (L3)
// This enables SLURM to schedule jobs across both GPU and CPU nodes.
// Nodes without IB labels are assigned to default CPU partition.
func buildInstanceTopology(node corev1.Node) *topology.InstanceTopology {
	hasTopology, labels := extractTopologyLabels(node.Labels)

	// CPU nodes: no IB labels, use defaults under common root
	if !hasTopology {
		klog.V(4).Infof("Node %q using CPU defaults", node.Name)
		return &topology.InstanceTopology{
			InstanceID:     node.Name,
			DatacenterID:   DefaultDatacenter,   // crusoe (common root)
			SpineID:        DefaultCPUPartition, // cpu-partition
			BlockID:        DefaultCPUPod,       // cpu-pod
			DatacenterName: DefaultDatacenter,
			SpineName:      DefaultCPUPartition,
			BlockName:      DefaultCPUPod,
			AcceleratorID:  "", // No IB for CPU
		}
	}

	// GPU nodes: have all 3 IB labels, placed under common root
	return &topology.InstanceTopology{
		InstanceID:     node.Name,
		DatacenterID:   DefaultDatacenter,                   // L1: crusoe (common root)
		SpineID:        labels.PartitionID,                  // L2: crusoe.ai/ib.partition.id
		BlockID:        labels.PodID,                        // L3: crusoe.ai/pod.id
		DatacenterName: DefaultDatacenter,                   // crusoe
		SpineName:      "partition-" + labels.PartitionName, // IB partition name
		BlockName:      "pod-" + labels.PodID,               // Pod UUID
		AcceleratorID:  labels.PartitionID,                  // IB partition = high-speed domain
	}
}

// resolveEmptyTopologyError determines the correct error based on why the result was empty
func resolveEmptyTopologyError(stats extractionStats) *httperr.Error {
	if stats.badLabels > 0 {
		return httperr.NewError(http.StatusInternalServerError,
			fmt.Sprintf("found matching nodes but %d were missing topology labels", stats.badLabels))
	}
	return httperr.NewError(http.StatusNotFound, "no requested nodes found in cluster")
}
