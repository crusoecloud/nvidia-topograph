# Crusoe Topology Provider

The Crusoe topology provider enables SLURM topology-aware scheduling for Crusoe Cloud clusters. It works with the **Crusoe Slurm Operator** offering to automatically discover and configure network topology based on InfiniBand partitions.

## How It Works

The provider reads topology information from Kubernetes node labels that are automatically applied to Crusoe Cloud nodes:

| Label | Level | Description |
|-------|-------|-------------|
| `crusoe.ai/ib.partition.id` | L2 (Spine) | InfiniBand partition identifier |
| `crusoe.ai/pod.id` | L3 (Block) | Pod (leaf switch) identifier |

This creates a **2-tier topology** with a common datacenter root 

### Topology Structure

```
topology/tree (Common Datacenter Root - enables cross-partition scheduling)
═══════════════════════════════════════════════════════════════════════════

crusoe                                     ← Common datacenter root for ALL nodes
├── partition-ibp-1                        ← GPU partition 1 (have IB labels)
│   ├── pod-pod-1 → [gpu-1, gpu-2, gpu-3, gpu-4]
│   └── pod-pod-2 → [gpu-5, gpu-6, gpu-7, gpu-8]
├── partition-ibp-2                        ← GPU partition 2 (have IB labels)
│   ├── pod-pod-3 → [gpu-9, gpu-10]
│   └── pod-pod-4 → [gpu-11, gpu-12]
└── partition-cpu-partition                ← CPU fallback (missing IB labels)
    └── pod-cpu-pod → [cpu-1, cpu-2, cpu-3, cpu-4]
```

### CPU Node Handling

Nodes missing either required IB label are automatically placed in a fallback partition:
- `DatacenterID`: `crusoe` (same as GPU nodes)
- `SpineID`: `cpu-partition`
- `BlockID`: `cpu-pod`

This ensures CPU nodes are visible to SLURM scheduling under the same datacenter root, enabling cross-partition job scheduling.

## Usage

When using the Crusoe Slurm Operator, topology discovery is configured automatically. No additional credentials or API keys are required because the provider reads network topology directly from Kubernetes node labels that are automatically managed by Crusoe CMK (Cloud Managed Kubernetes).

## Output Format

Example SLURM topology.conf output (with CPU nodes):

```
# datacenter-crusoe=crusoe
SwitchName=datacenter-crusoe Switches=partition-msi-h200-icat-ibp,partition-cpu-partition
# partition-msi-h200-icat-ibp=76034b3f-a826-4fb5-8a76-9afd8bc9fa8b
SwitchName=partition-msi-h200-icat-ibp Switches=pod-ca6b3558
# partition-cpu-partition=cpu-partition
SwitchName=partition-cpu-partition Switches=pod-cpu-pod
# pod-ca6b3558=ca6b3558-e3bf-fad1-2748-73582365f740
SwitchName=pod-ca6b3558 Nodes=test-cluster-h200-[0-1]
# pod-cpu-pod=cpu-pod
SwitchName=pod-cpu-pod Nodes=test-cluster-c1as-0
```

The hierarchy in this example:
- **L1 Datacenter**: `datacenter-crusoe` (common root for all nodes)
- **L2 Partition**: `partition-msi-h200-icat-ibp` (uses human-readable name from label) and `partition-cpu-partition` (CPU fallback)
- **L3 Pod**: `pod-ca6b3558` (truncated UUID) and `pod-cpu-pod` (CPU fallback)
