# Crusoe Provider

The Crusoe provider enables SLURM topology-aware scheduling for Crusoe Cloud clusters based on InfiniBand network partitions.

## Architecture

Crusoe uses a 2-tier topology:

- **Tier 1 (Partition)**: Physical IB partition boundary (`crusoe.ai/ib.partition.id`)
  - SLURM MUST NOT schedule jobs across different partitions (VMs cannot communicate)
- **Tier 2 (Pod)**: Leaf switch grouping (`crusoe.ai/pod.id`)
  - VMs on the same switch have better network performance due to lesser network hops

## Required Node Labels

The provider reads topology information from Kubernetes node labels:

| Label | Description | Example |
|-------|-------------|---------|
| `crusoe.ai/ib.partition.id` | IB partition UUID | `76034b3f-a826-4fb5-8a76-9afd8bc9fa8b` |
| `crusoe.ai/pod.id` | Pod (leaf switch) UUID | `ca6b3558-e3bf-fad1-2748-73582365f740` |

## Provider Modes

The Crusoe provider has two modes of operation:

### Production Mode (`crusoe`)

Uses live Kubernetes cluster data to build topology:

- **Provider name**: `crusoe`
- **Data source**: Kubernetes node labels via in-cluster or kubeconfig authentication
- **Use case**: Production deployments and integration testing with real clusters
- **Requirements**:
  - K8s cluster access (in-cluster service account or kubeconfig)
  - Nodes labeled with `crusoe.ai/ib.partition.id` and `crusoe.ai/pod.id`

```json
{
  "provider": {"name": "crusoe"},
  "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
}
```

### Simulation Mode (`crusoe-sim`)

Uses YAML model files to simulate topology without a real cluster:

- **Provider name**: `crusoe-sim`
- **Data source**: YAML model file (e.g., `tests/models/crusoe-small.yaml`)
- **Use case**: Unit tests, CI/CD pipelines, local development without cluster access
- **Requirements**: Valid YAML model file with topology structure

```json
{
  "provider": {
    "name": "crusoe-sim",
    "params": {"model_path": "tests/models/crusoe-small.yaml"}
  },
  "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
}
```

**When to use each mode**:

| Scenario | Mode |
|----------|------|
| Production deployment | `crusoe` (Production) |
| Integration testing with real K8s | `crusoe` (Production) |
| Unit tests / CI pipelines | `crusoe-sim` (Simulation) |
| Local development without cluster | `crusoe-sim` (Simulation) |
| Validating topology logic | `crusoe-sim` (Simulation) |

## Testing

### Unit Tests

```bash
# Run all Crusoe provider tests
go test -v ./pkg/providers/crusoe/...

# Run specific test
go test -v -run TestProviderSim ./pkg/providers/crusoe/...
```

### Local Testing with Real K8s Cluster

1. **Build and run topograph server**:
   ```bash
   go run ./cmd/topograph -v=2
   ```
   The `-v=2` flag enables verbose logging to see detailed node processing.

2. **Verify your nodes have required labels**:
   ```bash
   kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.crusoe\.ai/ib\.partition\.id}{"\t"}{.metadata.labels.crusoe\.ai/pod\.id}{"\n"}{end}'
   ```

3. **Send a topology request** (without filtering - all nodes):
   ```bash
   curl -X POST http://localhost:49021/v1/generate \
     -H "Content-Type: application/json" \
     -d '{
       "provider": {"name": "crusoe"},
       "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
     }'
   ```

4. **Send a topology request** (with specific nodes):
   ```bash
   curl -X POST http://localhost:49021/v1/generate \
     -H "Content-Type: application/json" \
     -d '{
       "provider": {"name": "crusoe"},
       "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}},
       "nodes": [
         {"region": "local", "instances": {"node-1": "node-1", "node-2": "node-2"}}
       ]
     }'
   ```

5. **Get the result**:
   ```bash
   curl http://localhost:49021/v1/topology/<request-id>
   ```

### Simulation Testing

Use the `crusoe-sim` provider to test without a real K8s cluster:

```bash
curl -X POST http://localhost:49021/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "provider": {
      "name": "crusoe-sim",
      "params": {"model_path": "tests/models/crusoe-small.yaml"}
    },
    "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
  }'
```

### Debugging

**Verbose logging levels**:
- `-v=2`: Shows requested node IDs, each node's labels, and skip/add decisions
- `-v=4`: Shows extracted topology labels for each node

**Common issues**:

1. **"no requested nodes found in cluster"**: The node names in your request don't match actual K8s node names. Check with `kubectl get nodes`.

2. **"missing labels"**: Nodes exist but are missing required `crusoe.ai/ib.partition.id` or `crusoe.ai/pod.id` labels.

3. **All nodes skipped**: When filtering is enabled, ensure the instance IDs in your request match the K8s node names exactly.

## Configuration

### Provider Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `nodeSelector` | `map[string]string` | Optional: Filter nodes by labels |

Example with node selector:
```json
{
  "provider": {
    "name": "crusoe",
    "params": {
      "nodeSelector": {"node.kubernetes.io/instance-type": "h100.8x"}
    }
  }
}
```

### RBAC Requirements

The provider needs read access to Kubernetes nodes:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: topograph-crusoe
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list"]
```

## Output Format

The provider generates SLURM topology configuration:

```
SwitchName=crusoe Switches=partition-1,partition-2
SwitchName=partition-1 Switches=pod-abc123,pod-def456
SwitchName=pod-abc123 Nodes=node-1,node-2,node-3,node-4
SwitchName=pod-def456 Nodes=node-5,node-6,node-7,node-8
```

This ensures SLURM schedules jobs within the same partition for proper IB connectivity.
