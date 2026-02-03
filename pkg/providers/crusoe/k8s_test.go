/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractTopologyLabels(t *testing.T) {
	testCases := []struct {
		name           string
		labels         map[string]string
		expectHasTopo  bool
		expectedLabels *TopologyLabels
	}{
		{
			name: "valid GPU labels with UUIDs (no partition name - uses full partition ID)",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				"crusoe.ai/pod.id":          "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
			expectHasTopo: true,
			expectedLabels: &TopologyLabels{
				PartitionID:   "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				PartitionName: "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				PodID:         "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
		},
		{
			name: "valid GPU labels with partition name label",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id":   "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				"crusoe.ai/ib.partition.name": "msi-h200-icat-ibp",
				"crusoe.ai/pod.id":            "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
			expectHasTopo: true,
			expectedLabels: &TopologyLabels{
				PartitionID:   "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				PartitionName: "msi-h200-icat-ibp", // Uses label value
				PodID:         "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
		},
		{
			name: "missing partition label - returns false, nil",
			labels: map[string]string{
				"crusoe.ai/pod.id": "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name: "missing pod label - returns false, nil",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
			},
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name:           "empty labels map - returns false, nil",
			labels:         map[string]string{},
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name:           "nil labels map - returns false, nil",
			labels:         nil,
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name: "empty partition value - returns false, nil",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "",
				"crusoe.ai/pod.id":          "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name: "empty pod value - returns false, nil",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				"crusoe.ai/pod.id":          "",
			},
			expectHasTopo:  false,
			expectedLabels: nil,
		},
		{
			name: "GPU labels with extra k8s labels",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id":        "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				"crusoe.ai/pod.id":                 "ca6b3558-e3bf-fad1-2748-73582365f740",
				"node.kubernetes.io/instance-type": "a100.8x",
				"topology.kubernetes.io/region":   "us-east-1",
			},
			expectHasTopo: true,
			expectedLabels: &TopologyLabels{
				PartitionID:   "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				PartitionName: "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				PodID:         "ca6b3558-e3bf-fad1-2748-73582365f740",
			},
		},
		{
			name: "different UUIDs for each tier",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "b2c3d4e5-5555-6666-7777-888899990000",
				"crusoe.ai/pod.id":          "c3d4e5f6-aaaa-bbbb-cccc-ddddeeeeffff",
			},
			expectHasTopo: true,
			expectedLabels: &TopologyLabels{
				PartitionID:   "b2c3d4e5-5555-6666-7777-888899990000",
				PartitionName: "b2c3d4e5-5555-6666-7777-888899990000",
				PodID:         "c3d4e5f6-aaaa-bbbb-cccc-ddddeeeeffff",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasTopo, labels := extractTopologyLabels(tc.labels)
			require.Equal(t, tc.expectHasTopo, hasTopo)
			require.Equal(t, tc.expectedLabels, labels)
		})
	}
}
