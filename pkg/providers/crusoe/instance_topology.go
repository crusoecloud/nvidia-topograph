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

		instance, err := buildInstanceTopology(node)
		if err != nil {
			klog.V(4).Infof("Node %q skipped: %v", node.Name, err)
			stats.badLabels++
			continue
		}

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

// buildInstanceTopology constructs topology object from node labels
func buildInstanceTopology(node corev1.Node) (*topology.InstanceTopology, error) {
	partition, podID, err := extractTopologyLabels(node.Labels)
	if err != nil {
		return nil, err
	}

	return &topology.InstanceTopology{
		InstanceID:     node.Name,
		DatacenterID:   partition,
		SpineID:        podID,
		BlockID:        "",  // Empty for 2-tier topology (partition -> pod -> nodes)
		DatacenterName: partition,
		SpineName:      "pod-" + podID,
		BlockName:      "",
	}, nil
}

// resolveEmptyTopologyError determines the correct error based on why the result was empty
func resolveEmptyTopologyError(stats extractionStats) *httperr.Error {
	if stats.badLabels > 0 {
		return httperr.NewError(http.StatusInternalServerError,
			fmt.Sprintf("found matching nodes but %d were missing topology labels", stats.badLabels))
	}
	return httperr.NewError(http.StatusNotFound, "no requested nodes found in cluster")
}
