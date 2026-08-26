/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package node_observer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewStatusInformer(t *testing.T) {
	ctx := context.TODO()
	trigger := &Trigger{
		NodeSelector: map[string]string{"key": "val"},
		PodSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"key": "val"},
		},
	}
	informer, err := NewStatusInformer(ctx, nil, trigger, nil)
	require.NoError(t, err)
	require.NotNil(t, informer.nodeFactory)
	require.NotNil(t, informer.podFactory)
}

func TestNodeUpdateAffectsTopology(t *testing.T) {
	const clique = "29d9a0b8-948d-4a61-8b9e-fbbbf06c521b.32766"

	node := func(labels map[string]string, ready corev1.ConditionStatus, rv string) *corev1.Node {
		return &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "gpu-1", Labels: labels, ResourceVersion: rv},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: ready}},
			},
		}
	}

	testCases := []struct {
		name    string
		oldNode *corev1.Node
		newNode *corev1.Node
		expect  bool
	}{
		{
			// The case this exists for: GPU Feature Discovery labels the clique
			// after the node registers, and that decides the node's block.
			name:    "gaining the clique label affects topology",
			oldNode: node(map[string]string{"compute": "true"}, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{"compute": "true", "nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "2"),
			expect:  true,
		},
		{
			name:    "moving to another clique affects topology",
			oldNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{"nvidia.com/gpu.clique": "ad92df3f-0583-4633-883a-37db088fc1a7.32766"}, corev1.ConditionTrue, "2"),
			expect:  true,
		},
		{
			name:    "losing the clique label affects topology",
			oldNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{}, corev1.ConditionTrue, "2"),
			expect:  true,
		},
		{
			// Heartbeats and condition flips arrive constantly. Requeueing on them
			// would regenerate topology without end.
			name:    "a readiness change alone does not affect topology",
			oldNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionFalse, "2"),
			expect:  false,
		},
		{
			name:    "a resync with no change does not affect topology",
			oldNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{"nvidia.com/gpu.clique": clique}, corev1.ConditionTrue, "1"),
			expect:  false,
		},
		{
			name:    "an unlabelled node that stays unlabelled does not affect topology",
			oldNode: node(nil, corev1.ConditionTrue, "1"),
			newNode: node(map[string]string{}, corev1.ConditionTrue, "2"),
			expect:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, nodeUpdateAffectsTopology(tc.oldNode, tc.newNode))
		})
	}
}
