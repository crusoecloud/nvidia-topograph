/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"fmt"

	"k8s.io/klog/v2"
)

const (
	// Label keys for Crusoe topology information
	labelPartitionID = "crusoe.ai/ib.partition.id"
	labelSwitchID    = "crusoe.ai/pod.id"

	// Default partition for nodes without IB labels (CPU nodes)
	DefaultCPUPartition = "cpu-partition"
	DefaultCPUPod       = "cpu-pod"
)

// extractTopologyLabels extracts partition_id and switch_id from node labels
func extractTopologyLabels(labels map[string]string) (partitionID, switchID string, err error) {
	partitionID, ok := labels[labelPartitionID]
	if !ok || len(partitionID) == 0 {
		return "", "", fmt.Errorf("missing or empty label %q", labelPartitionID)
	}

	switchID, ok = labels[labelSwitchID]
	if !ok || len(switchID) == 0 {
		return "", "", fmt.Errorf("missing or empty label %q", labelSwitchID)
	}

	klog.V(4).Infof("Extracted topology labels: partition=%q switch=%q", partitionID, switchID)
	return partitionID, switchID, nil
}
