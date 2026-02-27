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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/topograph/internal/httperr"
	"github.com/NVIDIA/topograph/pkg/config"
	"github.com/NVIDIA/topograph/pkg/topology"
)

func TestNewReconciler(t *testing.T) {
	cfg := &config.Config{
		Provider: "test-provider",
		Engine:   "test-engine",
	}

	r := NewReconciler(cfg, 5*time.Minute)

	require.Equal(t, 5*time.Minute, r.period)
	require.Equal(t, "test-provider", r.request.Provider.Name)
	require.Equal(t, "test-engine", r.request.Engine.Name)
	require.NotNil(t, r.processFn)
}

func TestReconcilerLifecycle(t *testing.T) {
	var callCount atomic.Int32

	cfg := &config.Config{
		Provider: "test-provider",
		Engine:   "test-engine",
	}

	r := NewReconciler(cfg, 50*time.Millisecond)
	r.processFn = func(item any) (any, *httperr.Error) {
		callCount.Add(1)
		tr := item.(*topology.Request)
		require.Equal(t, "test-provider", tr.Provider.Name)
		require.Equal(t, "test-engine", tr.Engine.Name)
		return nil, nil
	}

	done := make(chan error, 1)
	go func() {
		done <- r.Start()
	}()

	// Wait long enough for at least 3 invocations: 1 immediate + 2 ticks
	time.Sleep(130 * time.Millisecond)
	r.Stop(nil)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("reconciler did not stop in time")
	}

	require.GreaterOrEqual(t, int(callCount.Load()), 3)
}

func TestReconcilerContinuesOnError(t *testing.T) {
	var callCount atomic.Int32

	cfg := &config.Config{
		Provider: "test-provider",
		Engine:   "test-engine",
	}

	r := NewReconciler(cfg, 50*time.Millisecond)
	r.processFn = func(item any) (any, *httperr.Error) {
		callCount.Add(1)
		return nil, httperr.NewError(500, "transient error")
	}

	done := make(chan error, 1)
	go func() {
		done <- r.Start()
	}()

	// Allow multiple ticks even with errors
	time.Sleep(130 * time.Millisecond)
	r.Stop(nil)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("reconciler did not stop in time")
	}

	// Reconciler must keep running despite errors
	require.GreaterOrEqual(t, int(callCount.Load()), 3)
}
