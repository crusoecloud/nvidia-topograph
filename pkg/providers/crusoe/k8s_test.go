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
		name              string
		labels            map[string]string
		expectedPartition string
		expectedSwitch    string
		errMsg            string
	}{
		{
			name: "valid labels",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "partition-abc-123",
				"crusoe.ai/pod.id":          "pod-xyz-456",
			},
			expectedPartition: "partition-abc-123",
			expectedSwitch:    "pod-xyz-456",
		},
		{
			name: "missing partition label",
			labels: map[string]string{
				"crusoe.ai/pod.id": "pod-xyz-456",
			},
			errMsg: "missing or empty label \"crusoe.ai/ib.partition.id\"",
		},
		{
			name: "missing pod label",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "partition-abc-123",
			},
			errMsg: "missing or empty label \"crusoe.ai/pod.id\"",
		},
		{
			name:   "empty labels map",
			labels: map[string]string{},
			errMsg: "missing or empty label \"crusoe.ai/ib.partition.id\"",
		},
		{
			name:   "nil labels map",
			labels: nil,
			errMsg: "missing or empty label \"crusoe.ai/ib.partition.id\"",
		},
		{
			name: "empty partition value",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "",
				"crusoe.ai/pod.id":          "pod-xyz-456",
			},
			errMsg: "missing or empty label \"crusoe.ai/ib.partition.id\"",
		},
		{
			name: "empty pod value",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id": "partition-abc-123",
				"crusoe.ai/pod.id":          "",
			},
			errMsg: "missing or empty label \"crusoe.ai/pod.id\"",
		},
		{
			name: "with extra labels",
			labels: map[string]string{
				"crusoe.ai/ib.partition.id":            "partition-abc-123",
				"crusoe.ai/pod.id":                     "pod-xyz-456",
				"node.kubernetes.io/instance-type":    "a100.8x",
				"topology.kubernetes.io/region":       "us-east-1",
			},
			expectedPartition: "partition-abc-123",
			expectedSwitch:    "pod-xyz-456",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			partition, switchID, err := extractTopologyLabels(tc.labels)
			if tc.errMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPartition, partition)
				require.Equal(t, tc.expectedSwitch, switchID)
			}
		})
	}
}
