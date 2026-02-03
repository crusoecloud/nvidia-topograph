/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NVIDIA/topograph/pkg/topology"
)

func TestGetRequestedNodeIDs(t *testing.T) {
	testCases := []struct {
		name      string
		instances []topology.ComputeInstances
		expected  map[string]struct{}
		errCode   int
		errMsg    string
	}{
		{
			name:      "empty instances returns nil",
			instances: []topology.ComputeInstances{},
			expected:  nil,
		},
		{
			name: "single region with instances",
			instances: []topology.ComputeInstances{
				{
					Region:    "local",
					Instances: map[string]string{"node-1": "node-1", "node-2": "node-2"},
				},
			},
			expected: map[string]struct{}{"node-1": {}, "node-2": {}},
		},
		{
			name: "single region with empty instances",
			instances: []topology.ComputeInstances{
				{
					Region:    "local",
					Instances: map[string]string{},
				},
			},
			expected: nil,
		},
		{
			name: "multi-region returns error",
			instances: []topology.ComputeInstances{
				{Region: "region1", Instances: map[string]string{"node-1": "node-1"}},
				{Region: "region2", Instances: map[string]string{"node-2": "node-2"}},
			},
			errCode: http.StatusBadRequest,
			errMsg:  "Crusoe does not support multi-region topology requests",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getRequestedNodeIDs(tc.instances)
			if tc.errMsg != "" {
				require.NotNil(t, err)
				require.Equal(t, tc.errCode, err.Code())
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.Nil(t, err)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestShouldSkipNode(t *testing.T) {
	testCases := []struct {
		name         string
		nodeName     string
		requestedIDs map[string]struct{}
		expected     bool
	}{
		{
			name:         "nil requestedIDs returns false",
			nodeName:     "node-1",
			requestedIDs: nil,
			expected:     false,
		},
		{
			name:         "node in requestedIDs returns false",
			nodeName:     "node-1",
			requestedIDs: map[string]struct{}{"node-1": {}, "node-2": {}},
			expected:     false,
		},
		{
			name:         "node not in requestedIDs returns true",
			nodeName:     "node-3",
			requestedIDs: map[string]struct{}{"node-1": {}, "node-2": {}},
			expected:     true,
		},
		{
			name:         "empty requestedIDs skips all nodes",
			nodeName:     "node-1",
			requestedIDs: map[string]struct{}{},
			expected:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := shouldSkipNode(tc.nodeName, tc.requestedIDs)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildInstanceTopology(t *testing.T) {
	testCases := []struct {
		name     string
		node     corev1.Node
		expected *topology.InstanceTopology
	}{
		{
			name: "valid GPU node with all 3 labels (UUIDs)",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gpu-node-1",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id": "b2c3d4e5-5555-6666-7777-888899990000",
						"crusoe.ai/pod.id":          "c3d4e5f6-aaaa-bbbb-cccc-ddddeeeeffff",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "gpu-node-1",
				DatacenterID:   DefaultDatacenter,                                   // L1: crusoe (common root)
				SpineID:        "b2c3d4e5-5555-6666-7777-888899990000",              // L2: Partition
				BlockID:        "c3d4e5f6-aaaa-bbbb-cccc-ddddeeeeffff",              // L3: Pod
				DatacenterName: DefaultDatacenter,
				SpineName:      "partition-b2c3d4e5-5555-6666-7777-888899990000",
				BlockName:      "pod-c3d4e5f6-aaaa-bbbb-cccc-ddddeeeeffff",
				AcceleratorID:  "b2c3d4e5-5555-6666-7777-888899990000", // IB partition for high-speed domain
			},
		},
		{
			name: "GPU node with partition.name label",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gpu-node-3",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id":   "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
						"crusoe.ai/ib.partition.name": "msi-h200-icat-ibp",
						"crusoe.ai/pod.id":            "ca6b3558-e3bf-fad1-2748-73582365f740",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "gpu-node-3",
				DatacenterID:   DefaultDatacenter,
				SpineID:        "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
				BlockID:        "ca6b3558-e3bf-fad1-2748-73582365f740",
				DatacenterName: DefaultDatacenter,
				SpineName:      "partition-msi-h200-icat-ibp", // Uses full name label
				BlockName:      "pod-ca6b3558-e3bf-fad1-2748-73582365f740",
				AcceleratorID:  "76034b3f-a826-4fb5-8a76-9afd8bc9fa8b",
			},
		},
		{
			name: "node with both partition and pod labels (treated as GPU)",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gpu-node-2",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id": "partition-def",
						"crusoe.ai/pod.id":          "pod-123",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "gpu-node-2",
				DatacenterID:   DefaultDatacenter,
				SpineID:        "partition-def",                       // Uses partition ID
				BlockID:        "pod-123",                            // Uses pod ID
				DatacenterName: DefaultDatacenter,
				SpineName:      "partition-partition-def",           // Truncated
				BlockName:      "pod-pod-123",
				AcceleratorID:  "partition-def",                     // IB partition for high-speed domain
			},
		},
		{
			name: "missing partition label falls back to CPU partition",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cpu-node-2",
					Labels: map[string]string{
						"crusoe.ai/pod.id": "pod-123",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "cpu-node-2",
				DatacenterID:   DefaultDatacenter,
				SpineID:        DefaultCPUPartition,
				BlockID:        DefaultCPUPod,
				DatacenterName: DefaultDatacenter,
				SpineName:      DefaultCPUPartition,
				BlockName:      DefaultCPUPod,
				AcceleratorID:  "",
			},
		},
		{
			name: "missing pod label falls back to CPU partition",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cpu-node-3",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id": "partition-def",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "cpu-node-3",
				DatacenterID:   DefaultDatacenter,
				SpineID:        DefaultCPUPartition,
				BlockID:        DefaultCPUPod,
				DatacenterName: DefaultDatacenter,
				SpineName:      DefaultCPUPartition,
				BlockName:      DefaultCPUPod,
				AcceleratorID:  "",
			},
		},
		{
			name: "empty labels falls back to CPU partition",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "cpu-node-4",
					Labels: map[string]string{},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "cpu-node-4",
				DatacenterID:   DefaultDatacenter,
				SpineID:        DefaultCPUPartition,
				BlockID:        DefaultCPUPod,
				DatacenterName: DefaultDatacenter,
				SpineName:      DefaultCPUPartition,
				BlockName:      DefaultCPUPod,
				AcceleratorID:  "",
			},
		},
		{
			name: "nil labels falls back to CPU partition",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "cpu-node-5",
					Labels: nil,
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "cpu-node-5",
				DatacenterID:   DefaultDatacenter,
				SpineID:        DefaultCPUPartition,
				BlockID:        DefaultCPUPod,
				DatacenterName: DefaultDatacenter,
				SpineName:      DefaultCPUPartition,
				BlockName:      DefaultCPUPod,
				AcceleratorID:  "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildInstanceTopology(tc.node)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestResolveEmptyTopologyError(t *testing.T) {
	testCases := []struct {
		name    string
		stats   extractionStats
		errCode int
		errMsg  string
	}{
		{
			name:    "no nodes found",
			stats:   extractionStats{success: 0, skipped: 0, badLabels: 0},
			errCode: http.StatusNotFound,
			errMsg:  "no requested nodes found in cluster",
		},
		{
			name:    "nodes found but missing labels",
			stats:   extractionStats{success: 0, skipped: 0, badLabels: 5},
			errCode: http.StatusInternalServerError,
			errMsg:  "found matching nodes but 5 were missing topology labels",
		},
		{
			name:    "some skipped, some bad labels",
			stats:   extractionStats{success: 0, skipped: 3, badLabels: 2},
			errCode: http.StatusInternalServerError,
			errMsg:  "found matching nodes but 2 were missing topology labels",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := resolveEmptyTopologyError(tc.stats)
			require.NotNil(t, err)
			require.Equal(t, tc.errCode, err.Code())
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}
