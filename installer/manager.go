// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package installer provides installation management for skill dependencies.
//
// The installer package handles the installation and verification of
// external dependencies required by SKILL.md-defined skills. It supports
// multiple package managers and provides a unified interface for
// dependency resolution.
package installer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/plexusone/omniskill/loader"
)

// Manager handles installation of skill dependencies.
//
// Manager provides methods to verify, install, and manage external
// dependencies defined in SKILL.md files. It supports multiple
// installation methods including Go modules, npm packages, and pip.
type Manager struct {
	// Timeout is the maximum time for an installation command.
	// Zero means use default (5 minutes).
	Timeout time.Duration

	// Env contains additional environment variables for install commands.
	Env []string

	// DryRun when true, only reports what would be installed.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool

	// installers maps kind to installer function.
	installers map[string]InstallerFunc
}

// InstallerFunc is a function that installs a dependency.
type InstallerFunc func(ctx context.Context, step loader.InstallStep) error

// NewManager creates a new installation manager with default installers.
func NewManager() *Manager {
	m := &Manager{
		Timeout:    5 * time.Minute,
		installers: make(map[string]InstallerFunc),
	}

	// Register default installers
	m.RegisterInstaller("go", m.installGo)
	m.RegisterInstaller("npm", m.installNpm)
	m.RegisterInstaller("pip", m.installPip)
	m.RegisterInstaller("docker", m.installDocker)
	m.RegisterInstaller("brew", m.installBrew)

	return m
}

// RegisterInstaller registers a custom installer for a given kind.
func (m *Manager) RegisterInstaller(kind string, fn InstallerFunc) {
	m.installers[kind] = fn
}

// VerifyBinaries checks if all required binaries are available.
func (m *Manager) VerifyBinaries(bins []string) (missing []string) {
	for _, bin := range bins {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	return missing
}

// Install executes the installation steps for a skill.
func (m *Manager) Install(ctx context.Context, steps []loader.InstallStep) error {
	for _, step := range steps {
		if err := m.InstallStep(ctx, step); err != nil {
			return fmt.Errorf("install %s %s: %w", step.Kind, step.Module, err)
		}
	}
	return nil
}

// InstallStep executes a single installation step.
func (m *Manager) InstallStep(ctx context.Context, step loader.InstallStep) error {
	installer, ok := m.installers[step.Kind]
	if !ok {
		return fmt.Errorf("unsupported installer kind: %s", step.Kind)
	}

	if m.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.Timeout)
		defer cancel()
	}

	if m.DryRun {
		fmt.Printf("[dry-run] Would install %s via %s\n", step.Module, step.Kind)
		return nil
	}

	if err := installer(ctx, step); err != nil {
		return err
	}

	// Run post-install script if specified
	if step.Script != "" {
		if err := m.runScript(ctx, step.Script); err != nil {
			return fmt.Errorf("post-install script: %w", err)
		}
	}

	// Verify installed binaries
	if len(step.Bins) > 0 {
		if missing := m.VerifyBinaries(step.Bins); len(missing) > 0 {
			return fmt.Errorf("binaries not found after install: %v", missing)
		}
	}

	return nil
}

// InstallMissing installs only the dependencies whose binaries are missing.
func (m *Manager) InstallMissing(ctx context.Context, steps []loader.InstallStep) error {
	for _, step := range steps {
		// Check if any required binary is missing
		if len(step.Bins) > 0 {
			missing := m.VerifyBinaries(step.Bins)
			if len(missing) == 0 {
				// All binaries present, skip
				continue
			}
		}

		if err := m.InstallStep(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

// installGo installs a Go module using 'go install'.
func (m *Manager) installGo(ctx context.Context, step loader.InstallStep) error {
	cmd := exec.CommandContext(ctx, "go", "install", step.Module)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// installNpm installs an npm package globally using 'npm install -g'.
func (m *Manager) installNpm(ctx context.Context, step loader.InstallStep) error {
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", step.Module)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// installPip installs a Python package using 'pip install'.
func (m *Manager) installPip(ctx context.Context, step loader.InstallStep) error {
	// Try pip3 first, fall back to pip
	pipCmd := "pip3"
	if _, err := exec.LookPath("pip3"); err != nil {
		pipCmd = "pip"
	}

	cmd := exec.CommandContext(ctx, pipCmd, "install", step.Module)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// installDocker pulls a Docker image.
func (m *Manager) installDocker(ctx context.Context, step loader.InstallStep) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", step.Module)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// installBrew installs a package using Homebrew.
func (m *Manager) installBrew(ctx context.Context, step loader.InstallStep) error {
	cmd := exec.CommandContext(ctx, "brew", "install", step.Module)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// runCommand executes a command and captures output.
func (m *Manager) runCommand(cmd *exec.Cmd) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if m.Verbose {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	if m.Verbose && stdout.Len() > 0 {
		fmt.Println(stdout.String())
	}

	return nil
}

// runScript executes a shell script.
func (m *Manager) runScript(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(), m.Env...)
	return m.runCommand(cmd)
}

// InstallResult contains the result of an installation attempt.
type InstallResult struct {
	// Step is the installation step that was executed.
	Step loader.InstallStep

	// Success indicates if the installation succeeded.
	Success bool

	// Error contains any error that occurred.
	Error error

	// Duration is how long the installation took.
	Duration time.Duration
}

// InstallWithResults installs steps and returns detailed results.
func (m *Manager) InstallWithResults(ctx context.Context, steps []loader.InstallStep) []InstallResult {
	results := make([]InstallResult, len(steps))

	for i, step := range steps {
		start := time.Now()
		err := m.InstallStep(ctx, step)
		results[i] = InstallResult{
			Step:     step,
			Success:  err == nil,
			Error:    err,
			Duration: time.Since(start),
		}
	}

	return results
}
