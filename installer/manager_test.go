// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package installer

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omniskill/loader"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", m.Timeout, 5*time.Minute)
	}

	// Verify default installers are registered
	kinds := []string{"go", "npm", "pip", "docker", "brew"}
	for _, kind := range kinds {
		if _, ok := m.installers[kind]; !ok {
			t.Errorf("Installer %q not registered", kind)
		}
	}
}

func TestManager_VerifyBinaries(t *testing.T) {
	m := NewManager()

	// Test with binaries that should exist on most systems
	missing := m.VerifyBinaries([]string{"sh", "ls"})
	if len(missing) != 0 {
		t.Errorf("Expected no missing binaries for sh/ls, got %v", missing)
	}

	// Test with a binary that shouldn't exist
	missing = m.VerifyBinaries([]string{"nonexistent_binary_xyz123"})
	if len(missing) != 1 {
		t.Errorf("Expected 1 missing binary, got %d", len(missing))
	}
}

func TestManager_DryRun(t *testing.T) {
	m := NewManager()
	m.DryRun = true

	step := loader.InstallStep{
		Kind:   "go",
		Module: "example.com/test/pkg@latest",
	}

	// Dry run should succeed without actually installing
	err := m.InstallStep(context.Background(), step)
	if err != nil {
		t.Errorf("DryRun install failed: %v", err)
	}
}

func TestManager_UnsupportedKind(t *testing.T) {
	m := NewManager()

	step := loader.InstallStep{
		Kind:   "unsupported",
		Module: "some-module",
	}

	err := m.InstallStep(context.Background(), step)
	if err == nil {
		t.Error("Expected error for unsupported installer kind")
	}
}

func TestManager_RegisterInstaller(t *testing.T) {
	m := NewManager()

	called := false
	m.RegisterInstaller("custom", func(ctx context.Context, step loader.InstallStep) error {
		called = true
		return nil
	})

	step := loader.InstallStep{
		Kind:   "custom",
		Module: "test",
	}

	err := m.InstallStep(context.Background(), step)
	if err != nil {
		t.Errorf("Custom installer failed: %v", err)
	}

	if !called {
		t.Error("Custom installer was not called")
	}
}

func TestManager_InstallMissing(t *testing.T) {
	m := NewManager()
	m.DryRun = true

	steps := []loader.InstallStep{
		{
			Kind:   "go",
			Module: "example.com/test1@latest",
			Bins:   []string{"sh"}, // sh exists, so should skip
		},
		{
			Kind:   "go",
			Module: "example.com/test2@latest",
			Bins:   []string{"nonexistent_xyz"}, // doesn't exist, should install
		},
	}

	err := m.InstallMissing(context.Background(), steps)
	if err != nil {
		t.Errorf("InstallMissing failed: %v", err)
	}
}

func TestManager_InstallWithResults(t *testing.T) {
	m := NewManager()
	m.DryRun = true

	steps := []loader.InstallStep{
		{Kind: "go", Module: "example.com/test1@latest"},
		{Kind: "go", Module: "example.com/test2@latest"},
	}

	results := m.InstallWithResults(context.Background(), steps)

	if len(results) != 2 {
		t.Fatalf("Results length = %d, want 2", len(results))
	}

	for i, r := range results {
		if !r.Success {
			t.Errorf("Result[%d] failed: %v", i, r.Error)
		}
		if r.Duration < 0 {
			t.Errorf("Result[%d] has negative duration", i)
		}
	}
}

func TestManager_Timeout(t *testing.T) {
	m := NewManager()
	m.Timeout = 1 * time.Millisecond

	// Register a slow installer
	m.RegisterInstaller("slow", func(ctx context.Context, step loader.InstallStep) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	step := loader.InstallStep{
		Kind:   "slow",
		Module: "test",
	}

	err := m.InstallStep(context.Background(), step)
	if err == nil {
		t.Error("Expected timeout error")
	}
}
