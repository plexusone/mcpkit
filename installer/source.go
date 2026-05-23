// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SourceType identifies the type of skill source.
type SourceType string

const (
	// SourceTypeGit is a git repository URL.
	SourceTypeGit SourceType = "git"

	// SourceTypeLocal is a local filesystem path.
	SourceTypeLocal SourceType = "local"
)

// Source represents a skill source location.
type Source struct {
	// Type is the source type (git or local).
	Type SourceType

	// URL is the git repository URL (for git sources).
	URL string

	// Path is the local filesystem path (for local sources).
	Path string

	// Ref is the git reference (branch, tag, commit) to checkout.
	// Empty means default branch.
	Ref string

	// Subdir is an optional subdirectory within the repo/path.
	Subdir string
}

// ParseSource parses a source string into a Source struct.
// Formats:
//   - github.com/user/repo -> git source
//   - git@github.com:user/repo.git -> git source
//   - https://github.com/user/repo -> git source
//   - /path/to/skill -> local source
//   - ./relative/path -> local source
//   - github.com/user/repo@v1.0.0 -> git source with ref
//   - github.com/user/repo/subdir -> git source with subdir
func ParseSource(s string) (*Source, error) {
	// Check for local paths first
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
		absPath, err := filepath.Abs(s)
		if err != nil {
			return nil, fmt.Errorf("resolve path: %w", err)
		}
		return &Source{
			Type: SourceTypeLocal,
			Path: absPath,
		}, nil
	}

	// Check for explicit git URLs
	if strings.HasPrefix(s, "git@") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		url, ref := splitRef(s)
		return &Source{
			Type: SourceTypeGit,
			URL:  url,
			Ref:  ref,
		}, nil
	}

	// Assume github.com/user/repo format
	if strings.Contains(s, "/") {
		return parseGitHubSource(s)
	}

	return nil, fmt.Errorf("cannot determine source type for %q", s)
}

// parseGitHubSource parses a GitHub-style source string.
// Examples:
//   - github.com/user/repo
//   - github.com/user/repo@v1.0.0
//   - github.com/user/repo/subdir
//   - github.com/user/repo/subdir@v1.0.0
func parseGitHubSource(s string) (*Source, error) {
	// Split off @ref if present
	source, ref := splitRef(s)

	// Parse path components
	parts := strings.Split(source, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid git source: %s (expected host/user/repo)", s)
	}

	host := parts[0]
	user := parts[1]
	repo := parts[2]

	// Check for subdir (parts after repo)
	var subdir string
	if len(parts) > 3 {
		subdir = strings.Join(parts[3:], "/")
	}

	url := fmt.Sprintf("https://%s/%s/%s.git", host, user, repo)

	return &Source{
		Type:   SourceTypeGit,
		URL:    url,
		Ref:    ref,
		Subdir: subdir,
	}, nil
}

// splitRef splits a source string into path and ref components.
// Example: "github.com/user/repo@v1.0.0" -> ("github.com/user/repo", "v1.0.0")
func splitRef(s string) (path, ref string) {
	if idx := strings.LastIndex(s, "@"); idx > 0 {
		// Make sure @ is not part of a git@ URL
		if !strings.HasPrefix(s, "git@") || idx > 4 {
			return s[:idx], s[idx+1:]
		}
	}
	return s, ""
}

// SkillInstaller handles installing skills from various sources.
type SkillInstaller struct {
	// SkillsDir is the target directory for installed skills.
	SkillsDir string

	// GlobalDir is the global skills directory.
	GlobalDir string

	// UseGlobal indicates whether to install to global directory.
	UseGlobal bool

	// Symlink uses symlinks for local installs instead of copying.
	Symlink bool

	// Verbose enables verbose output.
	Verbose bool
}

// DefaultSkillsDir returns the default local skills directory.
func DefaultSkillsDir() string {
	return "skills"
}

// DefaultGlobalDir returns the default global skills directory.
func DefaultGlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".omniskill", "skills")
	}
	return filepath.Join(home, ".omniskill", "skills")
}

// NewSkillInstaller creates a new skill installer with defaults.
func NewSkillInstaller() *SkillInstaller {
	return &SkillInstaller{
		SkillsDir: DefaultSkillsDir(),
		GlobalDir: DefaultGlobalDir(),
	}
}

// TargetDir returns the target directory based on UseGlobal setting.
func (i *SkillInstaller) TargetDir() string {
	if i.UseGlobal {
		return i.GlobalDir
	}
	return i.SkillsDir
}

// Install installs a skill from the given source string.
func (i *SkillInstaller) Install(ctx context.Context, sourceStr string) (*InstalledSkill, error) {
	source, err := ParseSource(sourceStr)
	if err != nil {
		return nil, err
	}

	switch source.Type {
	case SourceTypeGit:
		return i.InstallGit(ctx, source)
	case SourceTypeLocal:
		return i.InstallLocal(ctx, source)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

// InstallGit installs a skill from a git repository.
func (i *SkillInstaller) InstallGit(ctx context.Context, source *Source) (*InstalledSkill, error) {
	// Determine skill name from URL
	name := extractRepoName(source.URL)
	if source.Subdir != "" {
		name = filepath.Base(source.Subdir)
	}

	targetDir := filepath.Join(i.TargetDir(), name)

	// Check if already installed
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("skill %q already installed at %s", name, targetDir)
	}

	// Ensure target parent directory exists
	if err := os.MkdirAll(i.TargetDir(), 0755); err != nil {
		return nil, fmt.Errorf("create skills directory: %w", err)
	}

	// Clone repository
	if i.Verbose {
		fmt.Printf("Cloning %s...\n", source.URL)
	}

	cloneDir := targetDir
	if source.Subdir != "" {
		// Clone to temp, then move subdir
		cloneDir = targetDir + ".tmp"
	}

	args := []string{"clone", "--depth", "1"}
	if source.Ref != "" {
		args = append(args, "--branch", source.Ref)
	}
	args = append(args, source.URL, cloneDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w\n%s", err, output)
	}

	// Handle subdir extraction
	if source.Subdir != "" {
		subdirPath := filepath.Join(cloneDir, source.Subdir)
		if _, err := os.Stat(subdirPath); err != nil {
			os.RemoveAll(cloneDir)
			return nil, fmt.Errorf("subdir %q not found in repository", source.Subdir)
		}

		// Move subdir to target
		if err := os.Rename(subdirPath, targetDir); err != nil {
			os.RemoveAll(cloneDir)
			return nil, fmt.Errorf("extract subdir: %w", err)
		}

		// Clean up clone dir
		os.RemoveAll(cloneDir)
	}

	return &InstalledSkill{
		Name:       name,
		Path:       targetDir,
		Source:     source,
		SourceType: SourceTypeGit,
		Global:     i.UseGlobal,
	}, nil
}

// InstallLocal installs a skill from a local path.
func (i *SkillInstaller) InstallLocal(ctx context.Context, source *Source) (*InstalledSkill, error) {
	// Verify source exists
	info, err := os.Stat(source.Path)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("source must be a directory: %s", source.Path)
	}

	// Determine skill name from path
	name := filepath.Base(source.Path)
	targetDir := filepath.Join(i.TargetDir(), name)

	// Check if already installed
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("skill %q already installed at %s", name, targetDir)
	}

	// Ensure target parent directory exists
	if err := os.MkdirAll(i.TargetDir(), 0755); err != nil {
		return nil, fmt.Errorf("create skills directory: %w", err)
	}

	if i.Symlink {
		// Create symlink
		if i.Verbose {
			fmt.Printf("Linking %s -> %s\n", targetDir, source.Path)
		}
		if err := os.Symlink(source.Path, targetDir); err != nil {
			return nil, fmt.Errorf("create symlink: %w", err)
		}
	} else {
		// Copy directory
		if i.Verbose {
			fmt.Printf("Copying %s to %s\n", source.Path, targetDir)
		}
		if err := copyDir(source.Path, targetDir); err != nil {
			return nil, fmt.Errorf("copy directory: %w", err)
		}
	}

	return &InstalledSkill{
		Name:       name,
		Path:       targetDir,
		Source:     source,
		SourceType: SourceTypeLocal,
		Global:     i.UseGlobal,
		Symlinked:  i.Symlink,
	}, nil
}

// Uninstall removes an installed skill.
func (i *SkillInstaller) Uninstall(name string) error {
	// Check local first
	localPath := filepath.Join(i.SkillsDir, name)
	if _, err := os.Stat(localPath); err == nil {
		return os.RemoveAll(localPath)
	}

	// Check global
	globalPath := filepath.Join(i.GlobalDir, name)
	if _, err := os.Stat(globalPath); err == nil {
		return os.RemoveAll(globalPath)
	}

	return fmt.Errorf("skill %q not found", name)
}

// List returns all installed skills.
func (i *SkillInstaller) List() ([]*InstalledSkill, error) {
	var skills []*InstalledSkill

	// List local skills
	localSkills, _ := i.listDir(i.SkillsDir, false)
	skills = append(skills, localSkills...)

	// List global skills
	globalSkills, _ := i.listDir(i.GlobalDir, true)
	skills = append(skills, globalSkills...)

	return skills, nil
}

func (i *SkillInstaller) listDir(dir string, global bool) ([]*InstalledSkill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []*InstalledSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		info, _ := os.Lstat(path)
		symlinked := info != nil && info.Mode()&os.ModeSymlink != 0

		skills = append(skills, &InstalledSkill{
			Name:      entry.Name(),
			Path:      path,
			Global:    global,
			Symlinked: symlinked,
		})
	}

	return skills, nil
}

// InstalledSkill contains information about an installed skill.
type InstalledSkill struct {
	// Name is the skill name (directory name).
	Name string

	// Path is the installed path.
	Path string

	// Source is the original source location.
	Source *Source

	// SourceType is git or local.
	SourceType SourceType

	// Global indicates if installed to global directory.
	Global bool

	// Symlinked indicates if the skill is symlinked (local installs).
	Symlinked bool
}

// extractRepoName extracts the repository name from a git URL.
func extractRepoName(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Get last path component
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Skip .git directory
		if strings.Contains(relPath, ".git") {
			return nil
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, info.Mode())
	})
}
