# Crusoe Topology Provider

The Crusoe topology provider enables SLURM topology-aware scheduling for Crusoe Cloud clusters. It works with the **Crusoe Slurm Operator** offering to automatically discover and configure network topology based on InfiniBand partitions.

## How It Works

The provider reads topology information from Kubernetes node labels that are automatically applied to Crusoe Cloud nodes:

- `crusoe.ai/ib.partition.id` - InfiniBand partition identifier
- `crusoe.ai/pod.id` - Pod (leaf switch) identifier

This creates a 2-tier topology where:
- Jobs are scheduled within the same partition for proper InfiniBand connectivity
- Nodes on the same leaf switch have optimal network performance

## Usage

When using the Crusoe Slurm Operator, topology discovery is configured automatically. No additional credentials or API keys are required because the provider reads network topology directly from Kubernetes node labels that are automatically managed by Crusoe CMK (Cloud Managed Kubernetes).

For manual requests:

```json
{
  "provider": {"name": "crusoe"},
  "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
}
```

## Requirements

- Crusoe Cloud cluster with Slurm Operator deployed
- Nodes labeled with `crusoe.ai/ib.partition.id` and `crusoe.ai/pod.id` (applied automatically by Crusoe)
