/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/topograph/pkg/engines/slurm"
	"github.com/NVIDIA/topograph/pkg/providers"
	"github.com/NVIDIA/topograph/pkg/topology"
)

func TestProviderSim(t *testing.T) {
	ctx := context.Background()

	// All 16 GPU nodes matching the diagram in crusoe-small.yaml
	allGPUInstances := []topology.ComputeInstances{
		{
			Region: "local",
			Instances: map[string]string{
				"vm-11": "vm-11", "vm-12": "vm-12", "vm-13": "vm-13", "vm-14": "vm-14",
				"vm-21": "vm-21", "vm-22": "vm-22", "vm-23": "vm-23", "vm-24": "vm-24",
				"vm-31": "vm-31", "vm-32": "vm-32", "vm-33": "vm-33", "vm-34": "vm-34",
				"vm-41": "vm-41", "vm-42": "vm-42", "vm-43": "vm-43", "vm-44": "vm-44",
			},
		},
	}

	// Mixed GPU + CPU instances
	mixedInstances := []topology.ComputeInstances{
		{
			Region: "local",
			Instances: map[string]string{
				// GPU nodes (have IB labels)
				"vm-11": "vm-11", "vm-12": "vm-12",
				"vm-21": "vm-21",
				// CPU nodes (no IB labels - fall back to cpu-network/cpu-partition/cpu-pod)
				"cpu-01": "cpu-01", "cpu-02": "cpu-02",
			},
		},
	}

	testCases := []struct {
		name      string
		modelFile string
		pageSize  *int
		instances []topology.ComputeInstances
		params    map[string]any
		topology  string
		err       string
	}{
		{
			name:      "Case 1: valid 3-tier topology with all GPU instances",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			params:    map[string]any{"plugin": "topology/tree"},
			instances: allGPUInstances,
			// Common datacenter "crusoe" enables cross-partition scheduling
			topology: `# crusoe=crusoe
SwitchName=crusoe Switches=partition-ibp-[1-2]
# partition-ibp-1=ibp-1
SwitchName=partition-ibp-1 Switches=pod-pod-[1-2]
# partition-ibp-2=ibp-2
SwitchName=partition-ibp-2 Switches=pod-pod-[3-4]
# pod-pod-1=pod-1
SwitchName=pod-pod-1 Nodes=vm-[11-14]
# pod-pod-2=pod-2
SwitchName=pod-pod-2 Nodes=vm-[21-24]
# pod-pod-3=pod-3
SwitchName=pod-pod-3 Nodes=vm-[31-34]
# pod-pod-4=pod-4
SwitchName=pod-pod-4 Nodes=vm-[41-44]
`,
		},
		{
			name:      "Case 1b: mixed GPU + CPU topology",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			params:    map[string]any{"plugin": "topology/tree"},
			instances: mixedInstances,
			// Common datacenter "crusoe" contains both GPU partitions and CPU partition
			topology: `# crusoe=crusoe
SwitchName=crusoe Switches=partition-cpu-partition,partition-ibp-1
# partition-cpu-partition=cpu-partition
SwitchName=partition-cpu-partition Switches=pod-cpu-pod
# partition-ibp-1=ibp-1
SwitchName=partition-ibp-1 Switches=pod-pod-[1-2]
# pod-cpu-pod=cpu-pod
SwitchName=pod-cpu-pod Nodes=cpu-[01-02]
# pod-pod-1=pod-1
SwitchName=pod-pod-1 Nodes=vm-[11-12]
# pod-pod-2=pod-2
SwitchName=pod-pod-2 Nodes=vm-21
`,
		},
		{
			name:      "Case 2: filter specific instances",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			params:    map[string]any{"plugin": "topology/tree"},
			instances: []topology.ComputeInstances{
				{
					Region: "local",
					Instances: map[string]string{
						"vm-11": "vm-11", "vm-12": "vm-12", "vm-21": "vm-21",
					},
				},
			},
			topology: `# crusoe=crusoe
SwitchName=crusoe Switches=partition-ibp-1
# partition-ibp-1=ibp-1
SwitchName=partition-ibp-1 Switches=pod-pod-[1-2]
# pod-pod-1=pod-1
SwitchName=pod-pod-1 Nodes=vm-[11-12]
# pod-pod-2=pod-2
SwitchName=pod-pod-2 Nodes=vm-21
`,
		},
		{
			name:      "Case 3: multi-region error",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			instances: []topology.ComputeInstances{
				{
					Region:    "region1",
					Instances: map[string]string{"vm-11": "vm-11"},
				},
				{
					Region:    "region2",
					Instances: map[string]string{"vm-21": "vm-21"},
				},
			},
			err: "Crusoe does not support multi-region topology requests",
		},
		{
			name:      "Case 4: no matching nodes",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			params:    map[string]any{"plugin": "topology/tree"},
			instances: []topology.ComputeInstances{
				{
					Region:    "local",
					Instances: map[string]string{"nonexistent-node": "nonexistent-node"},
				},
			},
			err: "no requested nodes found in cluster",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := providers.Config{
				Params: map[string]any{"model_path": tc.modelFile},
			}
			provider, httpErr := LoaderSim(ctx, cfg)
			if httpErr != nil {
				if len(tc.err) == 0 {
					require.Nil(t, httpErr)
				} else {
					require.Contains(t, httpErr.Error(), tc.err)
				}
				return
			}

			topo, httpErr := provider.GenerateTopologyConfig(ctx, tc.pageSize, tc.instances)
			if len(tc.err) != 0 {
				require.Contains(t, httpErr.Error(), tc.err)
			} else {
				require.Nil(t, httpErr)
				data, httpErr := slurm.GenerateOutput(ctx, topo, tc.params)
				require.Nil(t, httpErr)
				require.Equal(t, tc.topology, string(data))
			}
		})
	}
}

func TestProviderSim_GetComputeInstances(t *testing.T) {
	ctx := context.Background()

	cfg := providers.Config{
		Params: map[string]any{"model_path": "../../../tests/models/crusoe-small.yaml"},
	}
	provider, httpErr := LoaderSim(ctx, cfg)
	require.Nil(t, httpErr)

	providerSim, ok := provider.(*ProviderSim)
	require.True(t, ok)

	instances, httpErr := providerSim.GetComputeInstances(ctx)
	require.Nil(t, httpErr)
	require.Len(t, instances, 1)
	require.Equal(t, "local", instances[0].Region)
	require.Len(t, instances[0].Instances, 20) // 4 GPU switches * 4 VMs each + 4 CPU nodes
}
