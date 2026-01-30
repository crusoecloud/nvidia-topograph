/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetParameters(t *testing.T) {
	testCases := []struct {
		name     string
		params   map[string]any
		expected *Params
		err      string
	}{
		{
			name:   "Case 1: empty parameters uses default node selector",
			params: map[string]any{},
			expected: &Params{
				NodeSelector: map[string]string{
					"slurm.crusoe.ai/compute-node-type": "true",
				},
				nodeListOpt: &metav1.ListOptions{
					LabelSelector: "slurm.crusoe.ai/compute-node-type=true",
				},
			},
		},
		{
			name: "Case 2: with node selector",
			params: map[string]any{
				"nodeSelector": map[string]string{
					"node.kubernetes.io/instance-type": "a40.8x",
				},
			},
			expected: &Params{
				NodeSelector: map[string]string{
					"node.kubernetes.io/instance-type": "a40.8x",
				},
				nodeListOpt: &metav1.ListOptions{
					LabelSelector: "node.kubernetes.io/instance-type=a40.8x",
				},
			},
		},
		{
			name: "Case 3: with multiple node selector labels",
			params: map[string]any{
				"nodeSelector": map[string]string{
					"node.kubernetes.io/instance-type": "a40.8x",
					"topology.kubernetes.io/zone":      "us-east-1a",
				},
			},
			expected: &Params{
				NodeSelector: map[string]string{
					"node.kubernetes.io/instance-type": "a40.8x",
					"topology.kubernetes.io/zone":      "us-east-1a",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := getParameters(tc.params)
			if len(tc.err) != 0 {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected.NodeSelector, params.NodeSelector)
				if tc.expected.nodeListOpt != nil {
					require.NotNil(t, params.nodeListOpt)
					require.Equal(t, tc.expected.nodeListOpt.LabelSelector, params.nodeListOpt.LabelSelector)
				}
			}
		})
	}
}

func TestNamedLoader(t *testing.T) {
	name, loader := NamedLoader()
	require.Equal(t, "crusoe", name)
	require.NotNil(t, loader)
}

func TestNamedLoaderSim(t *testing.T) {
	name, loader := NamedLoaderSim()
	require.Equal(t, "crusoe-sim", name)
	require.NotNil(t, loader)
}
