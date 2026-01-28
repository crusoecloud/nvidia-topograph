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
		errMsg   string
	}{
		{
			name: "valid node with all labels",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id": "partition-abc",
						"crusoe.ai/pod.id":          "pod-123",
					},
				},
			},
			expected: &topology.InstanceTopology{
				InstanceID:     "node-1",
				DatacenterID:   "partition-abc",
				SpineID:        "pod-123",
				BlockID:        "",
				DatacenterName: "partition-abc",
				SpineName:      "pod-pod-123",
				BlockName:      "",
			},
		},
		{
			name: "missing partition label",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"crusoe.ai/pod.id": "pod-123",
					},
				},
			},
			errMsg: "missing or empty label",
		},
		{
			name: "missing pod label",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"crusoe.ai/ib.partition.id": "partition-abc",
					},
				},
			},
			errMsg: "missing or empty label",
		},
		{
			name: "empty labels",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "node-1",
					Labels: map[string]string{},
				},
			},
			errMsg: "missing or empty label",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildInstanceTopology(tc.node)
			if tc.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, result)
			}
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
