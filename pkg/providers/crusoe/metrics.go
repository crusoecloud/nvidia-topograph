/*
 * Copyright 2025 NVIDIA CORPORATION
 * SPDX-License-Identifier: Apache-2.0
 */

package crusoe

import (
	"github.com/prometheus/client_golang/prometheus"
)

var nodesProcessed = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "nodes_processed_total",
		Help:      "Total number of nodes processed",
		Subsystem: "topograph_crusoe",
	},
	[]string{"status"}, // success, skipped, missing_labels
)

func init() {
	prometheus.MustRegister(nodesProcessed)
}
