# Crusoe Provider

The Crusoe provider enables SLURM topology-aware scheduling for Crusoe Cloud clusters based on InfiniBand network partitions.

## Architecture

Crusoe uses a 2-tier topology:

- **Tier 1 (Partition)**: IB partition boundary (`crusoe.ai/ib.partition.id`) - jobs cannot cross partitions
- **Tier 2 (Pod)**: Leaf switch grouping (`crusoe.ai/pod.id`) - better performance within same pod

## Testing

> **Note**: The `slinky` engine requires in-cluster config (`rest.InClusterConfig()`) with no kubeconfig fallback. Test locally with simulation, then deploy to cluster for integration testing.

### Step 1: Unit Tests (Simulation)

Test the provider logic locally without a cluster:

```bash
# Run all Crusoe provider tests
go test -v ./pkg/providers/crusoe/...
```

Alternatively, you can run the simulator locally and manually test the provider:

```bash
# Start topograph locally
go run ./cmd/topograph -c config/topograph-config.yaml -v=2

# Test with simulation (returns request ID)
curl -X POST http://localhost:49021/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "provider": {"name": "crusoe-sim", "params": {"model_path": "tests/models/crusoe-small.yaml"}},
    "engine": {"name": "slurm", "params": {"plugin": "topology/tree"}}
  }'
  # <request-id>

# Get the result (replace <request-id> with the ID from above)
curl http://localhost:49021/v1/topology\?uid\=<request-id>
```

### Step 2: Integration Tests (Deploy to Cluster)

#### Build and Push Docker Image

```bash
# Build for Linux amd64 (cross-compile from Mac)
docker buildx build --platform linux/amd64 \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=amd64 \
  -t ghcr.io/$GITHUB_USERNAME/topograph:dev \
  -f ./Dockerfile . --load

# Push to GitHub Container Registry
docker push ghcr.io/$GITHUB_USERNAME/topograph:dev
```

#### Deploy Topograph

```bash
helm upgrade --install topograph charts/topograph -n slurm \
  --set image.repository=ghcr.io/$GITHUB_USERNAME/topograph \
  --set image.tag=dev \
  --set node-observer.image.repository=ghcr.io/$GITHUB_USERNAME/topograph \
  --set node-observer.image.tag=dev \
  --set node-data-broker.initc.enabled=true \
  --set node-data-broker.initc.image.repository=ghcr.io/$GITHUB_USERNAME/topograph \
  --set node-data-broker.initc.image.tag=dev \
  --set global.provider.name=crusoe \
  --set global.engine.name=slinky \
  --set global.engine.params.namespace=slurm \
  --set 'global.engine.params.podSelector.matchLabels.app\.kubernetes\.io/component=worker' \
  --set global.engine.params.topologyConfigmapName=slurm-topology \
  --set global.engine.params.topologyConfigPath=topology.conf \
  --set global.engine.params.plugin=topology/tree
```

> **Note**: The node-data-broker adds `topograph.nvidia.com/instance` and `topograph.nvidia.com/region` annotations to nodes, which the slinky engine requires. For Crusoe, the init container sets instance ID = node name and region = "local".

#### Verification

In production, topology updates are triggered automatically by the **node-observer** component. When worker pods are added or removed, node-observer detects the change and sends a request to topograph.

```bash
# Watch topograph logs
kubectl logs -n slurm -l app.kubernetes.io/name=topograph -f

# Check the topology configuration added by topograph
kubectl get configmap slurm-topology -n slurm -o yaml
```

## Configuration

### Provider Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `nodeSelector` | `map[string]string` | Optional: Filter nodes by labels |

### Required Node Labels

| Label | Description |
|-------|-------------|
| `crusoe.ai/ib.partition.id` | IB partition UUID |
| `crusoe.ai/pod.id` | Pod (leaf switch) UUID |

## Output Format

```
SwitchName=partition-1 Switches=pod-abc123,pod-def456
SwitchName=pod-abc123 Nodes=node-1,node-2,node-3,node-4
SwitchName=pod-def456 Nodes=node-5,node-6,node-7,node-8
```

## Troubleshooting

### Docker build fails on Mac

Error:
```
/usr/local/go/pkg/tool/linux_amd64/compile: signal: segmentation fault (core dumped)
```

Add the platfrom to Dockerfile:

`FROM --platform=linux/arm64 golang:1.24.7 AS builder`
