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
	labelPartitionID = "crusoe.ai/partition-id"
	labelPodID       = "crusoe.ai/pod-id"
)

// extractTopologyLabels extracts partition_id and pod_id from node labels
func extractTopologyLabels(labels map[string]string) (partitionID, podID string, err error) {
	partitionID, ok := labels[labelPartitionID]
	if !ok || len(partitionID) == 0 {
		return "", "", fmt.Errorf("missing or empty label %q", labelPartitionID)
	}

	podID, ok = labels[labelPodID]
	if !ok || len(podID) == 0 {
		return "", "", fmt.Errorf("missing or empty label %q", labelPodID)
	}

	klog.V(4).Infof("Extracted topology labels: partition=%q pod=%q", partitionID, podID)
	return partitionID, podID, nil
}
