/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"k8s.io/klog/v2"
)

const (
	// Label keys for Crusoe topology information (2-tier hierarchy)
	labelPartitionID   = "crusoe.ai/ib.partition.id"        // L2: IB Partition
	labelPartitionName = "crusoe.ai/ib.partition.name"      // L2: IB Partition display name (optional)
	labelSwitchID      = "crusoe.ai/pod.id"                 // L3: Pod/Leaf switch

	// Common root datacenter for all nodes (enables cross-partition scheduling)
	DefaultDatacenter = "crusoe"

	// Default partition for nodes without IB labels (CPU nodes)
	DefaultCPUPartition = "cpu-partition"
	DefaultCPUPod       = "cpu-pod"
)

// TopologyLabels contains the extracted Crusoe topology labels
type TopologyLabels struct {
	PartitionID   string // L2: crusoe.ai/ib.partition.id
	PartitionName string // L2: crusoe.ai/ib.partition.name (falls back to truncated ID if not set)
	PodID         string // L3: crusoe.ai/pod.id
}

// extractTopologyLabels extracts topology levels from node labels (partition, pod)
// Returns (false, nil) if any required IB label is missing
// Returns (true, labels) with PartitionName falling back to PartitionID if label not set
func extractTopologyLabels(labels map[string]string) (bool, *TopologyLabels) {
	partitionID := labels[labelPartitionID]
	partitionName := labels[labelPartitionName]
	podID := labels[labelSwitchID]

	// GPU nodes must have both required labels
	if partitionID == "" || podID == "" {
		klog.V(4).Infof("Missing IB labels (partition=%q pod=%q)",
			partitionID, podID)
		return false, nil
	}

	// Fallback: if partition.name label not set, use full partition ID
	if partitionName == "" {
		partitionName = partitionID
	}

	klog.V(4).Infof("Extracted GPU topology labels: partition=%q name=%q pod=%q",
		partitionID, partitionName, podID)
	return true, &TopologyLabels{
		PartitionID:   partitionID,
		PartitionName: partitionName, // Either label value or truncated ID
		PodID:         podID,
	}
}
