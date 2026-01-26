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

	// Helper to create instances map with all 16 nodes
	allInstances := []topology.ComputeInstances{
		{
			Region: "local",
			Instances: map[string]string{
				"a1b2c3d4-1111-1111-1111-111111111111": "a1b2c3d4-1111-1111-1111-111111111111",
				"a1b2c3d4-1111-1111-1111-111111111112": "a1b2c3d4-1111-1111-1111-111111111112",
				"a1b2c3d4-1111-1111-1111-111111111113": "a1b2c3d4-1111-1111-1111-111111111113",
				"a1b2c3d4-1111-1111-1111-111111111114": "a1b2c3d4-1111-1111-1111-111111111114",
				"a1b2c3d4-2222-2222-2222-222222222221": "a1b2c3d4-2222-2222-2222-222222222221",
				"a1b2c3d4-2222-2222-2222-222222222222": "a1b2c3d4-2222-2222-2222-222222222222",
				"a1b2c3d4-2222-2222-2222-222222222223": "a1b2c3d4-2222-2222-2222-222222222223",
				"a1b2c3d4-2222-2222-2222-222222222224": "a1b2c3d4-2222-2222-2222-222222222224",
				"a1b2c3d4-3333-3333-3333-333333333331": "a1b2c3d4-3333-3333-3333-333333333331",
				"a1b2c3d4-3333-3333-3333-333333333332": "a1b2c3d4-3333-3333-3333-333333333332",
				"a1b2c3d4-3333-3333-3333-333333333333": "a1b2c3d4-3333-3333-3333-333333333333",
				"a1b2c3d4-3333-3333-3333-333333333334": "a1b2c3d4-3333-3333-3333-333333333334",
				"a1b2c3d4-4444-4444-4444-444444444441": "a1b2c3d4-4444-4444-4444-444444444441",
				"a1b2c3d4-4444-4444-4444-444444444442": "a1b2c3d4-4444-4444-4444-444444444442",
				"a1b2c3d4-4444-4444-4444-444444444443": "a1b2c3d4-4444-4444-4444-444444444443",
				"a1b2c3d4-4444-4444-4444-444444444444": "a1b2c3d4-4444-4444-4444-444444444444",
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
			name:      "Case 1: valid 2-partition topology with all instances",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			params:    map[string]any{"plugin": "topology/tree"},
			instances: allInstances,
			topology: `# 76034b3f-a826-4fb5-8a76-9afd8bc9fa8b=76034b3f-a826-4fb5-8a76-9afd8bc9fa8b
SwitchName=76034b3f-a826-4fb5-8a76-9afd8bc9fa8b Switches=pod-ca6b3558-e3bf-fad1-2748-73582365f[740-741]
# 76034b3f-a826-4fb5-8a76-9afd8bc9fa8c=76034b3f-a826-4fb5-8a76-9afd8bc9fa8c
SwitchName=76034b3f-a826-4fb5-8a76-9afd8bc9fa8c Switches=pod-ca6b3558-e3bf-fad1-2748-73582365f[742-743]
# pod-ca6b3558-e3bf-fad1-2748-73582365f740=ca6b3558-e3bf-fad1-2748-73582365f740
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f740 Nodes=a1b2c3d4-1111-1111-1111-[111111111111-111111111114]
# pod-ca6b3558-e3bf-fad1-2748-73582365f741=ca6b3558-e3bf-fad1-2748-73582365f741
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f741 Nodes=a1b2c3d4-2222-2222-2222-[222222222221-222222222224]
# pod-ca6b3558-e3bf-fad1-2748-73582365f742=ca6b3558-e3bf-fad1-2748-73582365f742
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f742 Nodes=a1b2c3d4-3333-3333-3333-[333333333331-333333333334]
# pod-ca6b3558-e3bf-fad1-2748-73582365f743=ca6b3558-e3bf-fad1-2748-73582365f743
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f743 Nodes=a1b2c3d4-4444-4444-4444-[444444444441-444444444444]
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
						"a1b2c3d4-1111-1111-1111-111111111111": "a1b2c3d4-1111-1111-1111-111111111111",
						"a1b2c3d4-1111-1111-1111-111111111112": "a1b2c3d4-1111-1111-1111-111111111112",
						"a1b2c3d4-2222-2222-2222-222222222221": "a1b2c3d4-2222-2222-2222-222222222221",
					},
				},
			},
			topology: `# 76034b3f-a826-4fb5-8a76-9afd8bc9fa8b=76034b3f-a826-4fb5-8a76-9afd8bc9fa8b
SwitchName=76034b3f-a826-4fb5-8a76-9afd8bc9fa8b Switches=pod-ca6b3558-e3bf-fad1-2748-73582365f[740-741]
# pod-ca6b3558-e3bf-fad1-2748-73582365f740=ca6b3558-e3bf-fad1-2748-73582365f740
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f740 Nodes=a1b2c3d4-1111-1111-1111-[111111111111-111111111112]
# pod-ca6b3558-e3bf-fad1-2748-73582365f741=ca6b3558-e3bf-fad1-2748-73582365f741
SwitchName=pod-ca6b3558-e3bf-fad1-2748-73582365f741 Nodes=a1b2c3d4-2222-2222-2222-222222222221
`,
		},
		{
			name:      "Case 3: multi-region error",
			modelFile: "../../../tests/models/crusoe-small.yaml",
			instances: []topology.ComputeInstances{
				{
					Region:    "region1",
					Instances: map[string]string{"a1b2c3d4-1111-1111-1111-111111111111": "node1"},
				},
				{
					Region:    "region2",
					Instances: map[string]string{"a1b2c3d4-2222-2222-2222-222222222221": "node2"},
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
	require.Len(t, instances[0].Instances, 16) // 4 switches * 4 VMs each
}
