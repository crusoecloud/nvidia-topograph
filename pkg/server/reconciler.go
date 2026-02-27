/*
 * Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package server

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/NVIDIA/topograph/internal/httperr"
	"github.com/NVIDIA/topograph/pkg/config"
	"github.com/NVIDIA/topograph/pkg/topology"
)

// Reconciler periodically regenerates the topology using the configured provider and engine.
type Reconciler struct {
	period    time.Duration
	request   *topology.Request
	stopCh    chan struct{}
	processFn func(any) (any, *httperr.Error)
}

// NewReconciler creates a Reconciler from the given config. The reconciler uses the
// top-level provider and engine names along with the reconciler-specific params.
func NewReconciler(cfg *config.Config, period time.Duration) *Reconciler {
	return &Reconciler{
		period: period,
		request: &topology.Request{
			Provider: topology.Provider{Name: cfg.Provider},
			Engine:   topology.Engine{Name: cfg.Engine},
		},
		stopCh:    make(chan struct{}),
		processFn: processRequest,
	}
}

// Start runs the reconcile loop. It regenerates the topology immediately on startup,
// then repeats on every tick of the configured period. Start blocks until Stop is called.
func (r *Reconciler) Start() error {
	klog.Infof("Starting reconciler with period %s", r.period)

	r.reconcile()

	ticker := time.NewTicker(r.period)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			klog.Info("Reconciler stopped")
			return nil
		case <-ticker.C:
			r.reconcile()
		}
	}
}

// Stop signals the reconciler to shut down.
func (r *Reconciler) Stop(err error) {
	klog.Infof("Stopping reconciler: %v", err)
	close(r.stopCh)
}

func (r *Reconciler) reconcile() {
	klog.Info("Reconciler: regenerating topology")
	if _, herr := r.processFn(r.request); herr != nil {
		klog.Errorf("Reconciler: failed to generate topology: %v", herr)
	} else {
		klog.Info("Reconciler: topology regenerated successfully")
	}
}
