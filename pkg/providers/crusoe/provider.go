/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"context"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/NVIDIA/topograph/internal/config"
	"github.com/NVIDIA/topograph/internal/httperr"
	"github.com/NVIDIA/topograph/pkg/providers"
	"github.com/NVIDIA/topograph/pkg/topology"
)

const NAME = "crusoe"

// baseProvider contains shared logic for both Provider and ProviderSim
type baseProvider struct {
	kubeClient *kubernetes.Clientset
	kubeConfig *rest.Config
	params     *Params
}

type Provider struct {
	baseProvider
}

// Params holds optional configuration for filtering K8s nodes
type Params struct {
	// NodeSelector (optional) filters nodes by labels (e.g., {"crusoe.ai/gpu": "h100"})
	NodeSelector map[string]string `mapstructure:"nodeSelector"`

	// nodeListOpt is derived from NodeSelector for K8s API calls
	nodeListOpt *metav1.ListOptions
}

func NamedLoader() (string, providers.Loader) {
	return NAME, Loader
}

// Loader creates provider using in-cluster K8s service account auth
func Loader(ctx context.Context, config providers.Config) (providers.Provider, *httperr.Error) {
	p, err := getParameters(config.Params)
	if err != nil {
		return nil, httperr.NewError(http.StatusBadRequest, err.Error())
	}

	// Use in-cluster config (no credentials needed, uses K8s service account)
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, httperr.NewError(http.StatusBadGateway, fmt.Sprintf("failed to get in-cluster config: %v", err))
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, httperr.NewError(http.StatusBadGateway, fmt.Sprintf("failed to create kubernetes client: %v", err))
	}

	klog.Infof("Created Crusoe provider with in-cluster Kubernetes client")

	return New(client, cfg, p), nil
}

// getParameters parses config and converts NodeSelector to K8s ListOptions
func getParameters(params map[string]any) (*Params, error) {
	p := &Params{}
	if err := config.Decode(params, p); err != nil {
		return nil, err
	}

	// Convert NodeSelector map to K8s label selector string for API calls
	if len(p.NodeSelector) != 0 {
		p.nodeListOpt = &metav1.ListOptions{
			LabelSelector: labels.Set(p.NodeSelector).String(),
		}
		klog.Infof("Using node selector: %v", p.NodeSelector)
	}

	return p, nil
}

func New(client *kubernetes.Clientset, config *rest.Config, params *Params) *Provider {
	return &Provider{
		baseProvider: baseProvider{
			kubeClient: client,
			kubeConfig: config,
			params:     params,
		},
	}
}

// GenerateTopologyConfig builds 2-tier Crusoe topology: partition → pod → instances
func (p *baseProvider) GenerateTopologyConfig(ctx context.Context, pageSize *int, instances []topology.ComputeInstances) (*topology.Vertex, *httperr.Error) {
	topo, err := p.generateInstanceTopology(ctx, instances)
	if err != nil {
		return nil, err
	}

	klog.Infof("Extracted topology for %d instances", topo.Len())

	// Convert to 3-tier graph (BlockID = SpineID for 2-tier systems)
	return topo.ToThreeTierGraph(NAME, instances, false), nil
}

// Engine support methods for SLURM integration

// Instances2NodeMap maps instance IDs to K8s node names (identity mapping for Crusoe)
func (p *Provider) Instances2NodeMap(ctx context.Context, nodes []string) (map[string]string, error) {
	// For Crusoe, instance ID = node name in Kubernetes
	result := make(map[string]string)
	for _, node := range nodes {
		result[node] = node
	}
	return result, nil
}

// GetInstancesRegions returns region for each instance (always "local" for Crusoe)
func (p *Provider) GetInstancesRegions(ctx context.Context, nodes []string) (map[string]string, error) {
	// Crusoe is single-region per cluster
	result := make(map[string]string)
	for _, node := range nodes {
		result[node] = "local"
	}
	return result, nil
}

// GetNodeAnnotations returns annotations required by slinky engine.
// For Crusoe, instance ID = node name and region = "local".
// Topology info comes from K8s labels (crusoe.ai/ib.partition.id, crusoe.ai/pod.id).
func GetNodeAnnotations(ctx context.Context, nodeName string) (map[string]string, error) {
	return map[string]string{
		topology.KeyNodeInstance: nodeName,
		topology.KeyNodeRegion:   "local",
	}, nil
}

