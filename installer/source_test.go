// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSource_Local(t *testing.T) {
	tests := []struct {
		input string
		want  SourceType
	}{
		{"/absolute/path/to/skill", SourceTypeLocal},
		{"./relative/path", SourceTypeLocal},
		{"../parent/path", SourceTypeLocal},
	}

	for _, tt := range tests {
		source, err := ParseSource(tt.input)
		if err != nil {
			t.Errorf("ParseSource(%q) failed: %v", tt.input, err)
			continue
		}
		if source.Type != tt.want {
			t.Errorf("ParseSource(%q).Type = %q, want %q", tt.input, source.Type, tt.want)
		}
	}
}

func TestParseSource_Git(t *testing.T) {
	tests := []struct {
		input   string
		wantURL string
		wantRef string
	}{
		{
			input:   "github.com/user/repo",
			wantURL: "https://github.com/user/repo.git",
			wantRef: "",
		},
		{
			input:   "github.com/user/repo@v1.0.0",
			wantURL: "https://github.com/user/repo.git",
			wantRef: "v1.0.0",
		},
		{
			input:   "https://github.com/user/repo",
			wantURL: "https://github.com/user/repo",
			wantRef: "",
		},
		{
			input:   "https://github.com/user/repo@main",
			wantURL: "https://github.com/user/repo",
			wantRef: "main",
		},
	}

	for _, tt := range tests {
		source, err := ParseSource(tt.input)
		if err != nil {
			t.Errorf("ParseSource(%q) failed: %v", tt.input, err)
			continue
		}
		if source.Type != SourceTypeGit {
			t.Errorf("ParseSource(%q).Type = %q, want %q", tt.input, source.Type, SourceTypeGit)
		}
		if source.URL != tt.wantURL {
			t.Errorf("ParseSource(%q).URL = %q, want %q", tt.input, source.URL, tt.wantURL)
		}
		if source.Ref != tt.wantRef {
			t.Errorf("ParseSource(%q).Ref = %q, want %q", tt.input, source.Ref, tt.wantRef)
		}
	}
}

func TestParseSource_GitSubdir(t *testing.T) {
	source, err := ParseSource("github.com/user/repo/skills/weather")
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}

	if source.URL != "https://github.com/user/repo.git" {
		t.Errorf("URL = %q, want %q", source.URL, "https://github.com/user/repo.git")
	}
	if source.Subdir != "skills/weather" {
		t.Errorf("Subdir = %q, want %q", source.Subdir, "skills/weather")
	}
}

func TestParseSource_Invalid(t *testing.T) {
	_, err := ParseSource("invalid")
	if err == nil {
		t.Error("Expected error for invalid source")
	}
}

func TestSplitRef(t *testing.T) {
	tests := []struct {
		input    string
		wantPath string
		wantRef  string
	}{
		{"path@ref", "path", "ref"},
		{"path", "path", ""},
		{"git@github.com:user/repo", "git@github.com:user/repo", ""},
		{"git@github.com:user/repo@v1.0.0", "git@github.com:user/repo", "v1.0.0"},
	}

	for _, tt := range tests {
		path, ref := splitRef(tt.input)
		if path != tt.wantPath || ref != tt.wantRef {
			t.Errorf("splitRef(%q) = (%q, %q), want (%q, %q)",
				tt.input, path, ref, tt.wantPath, tt.wantRef)
		}
	}
}

func TestSkillInstaller_InstallLocal(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source-skill")
	targetDir := filepath.Join(tmpDir, "installed")

	// Create source skill with SKILL.md
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test"), 0600); err != nil {
		t.Fatal(err)
	}

	installer := &SkillInstaller{
		SkillsDir: targetDir,
		GlobalDir: filepath.Join(tmpDir, "global"),
	}

	source := &Source{
		Type: SourceTypeLocal,
		Path: sourceDir,
	}

	result, err := installer.InstallLocal(context.Background(), source)
	if err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	if result.Name != "source-skill" {
		t.Errorf("Name = %q, want %q", result.Name, "source-skill")
	}

	// Verify SKILL.md was copied
	if _, err := os.Stat(filepath.Join(result.Path, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found in installed skill")
	}
}

func TestSkillInstaller_InstallLocal_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source-skill")
	targetDir := filepath.Join(tmpDir, "installed")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test"), 0600); err != nil {
		t.Fatal(err)
	}

	installer := &SkillInstaller{
		SkillsDir: targetDir,
		Symlink:   true,
	}

	source := &Source{
		Type: SourceTypeLocal,
		Path: sourceDir,
	}

	result, err := installer.InstallLocal(context.Background(), source)
	if err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	if !result.Symlinked {
		t.Error("Expected Symlinked = true")
	}

	// Verify it's actually a symlink
	info, err := os.Lstat(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected symlink, got regular directory")
	}
}

func TestSkillInstaller_List(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	globalDir := filepath.Join(tmpDir, "global")

	// Create some skill directories
	os.MkdirAll(filepath.Join(localDir, "skill1"), 0755)
	os.MkdirAll(filepath.Join(localDir, "skill2"), 0755)
	os.MkdirAll(filepath.Join(globalDir, "skill3"), 0755)

	installer := &SkillInstaller{
		SkillsDir: localDir,
		GlobalDir: globalDir,
	}

	skills, err := installer.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("Found %d skills, want 3", len(skills))
	}

	// Check for global skill
	var hasGlobal bool
	for _, s := range skills {
		if s.Name == "skill3" && s.Global {
			hasGlobal = true
		}
	}
	if !hasGlobal {
		t.Error("Global skill not found or not marked as global")
	}
}

func TestSkillInstaller_Uninstall(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")

	skillDir := filepath.Join(localDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test"), 0644)

	installer := &SkillInstaller{
		SkillsDir: localDir,
		GlobalDir: filepath.Join(tmpDir, "global"),
	}

	err := installer.Uninstall("test-skill")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("Skill directory still exists after uninstall")
	}
}

func TestSkillInstaller_Uninstall_NotFound(t *testing.T) {
	installer := &SkillInstaller{
		SkillsDir: "/nonexistent",
		GlobalDir: "/also-nonexistent",
	}

	err := installer.Uninstall("missing-skill")
	if err == nil {
		t.Error("Expected error for missing skill")
	}
}

func TestSkillInstaller_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source-skill")
	targetDir := filepath.Join(tmpDir, "installed")

	// Create source and already-installed target
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(filepath.Join(targetDir, "source-skill"), 0755)

	installer := &SkillInstaller{
		SkillsDir: targetDir,
	}

	source := &Source{
		Type: SourceTypeLocal,
		Path: sourceDir,
	}

	_, err := installer.InstallLocal(context.Background(), source)
	if err == nil {
		t.Error("Expected error for already installed skill")
	}
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/myskill.git", "myskill"},
		{"https://github.com/user/myskill", "myskill"},
		{"git@github.com:user/repo.git", "repo"},
	}

	for _, tt := range tests {
		got := extractRepoName(tt.url)
		if got != tt.want {
			t.Errorf("extractRepoName(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestDefaultDirs(t *testing.T) {
	local := DefaultSkillsDir()
	if local != "skills" {
		t.Errorf("DefaultSkillsDir() = %q, want %q", local, "skills")
	}

	global := DefaultGlobalDir()
	if global == "" {
		t.Error("DefaultGlobalDir() returned empty string")
	}
}

func TestSkillInstaller_TargetDir(t *testing.T) {
	installer := &SkillInstaller{
		SkillsDir: "/local",
		GlobalDir: "/global",
	}

	if installer.TargetDir() != "/local" {
		t.Errorf("TargetDir() = %q, want /local", installer.TargetDir())
	}

	installer.UseGlobal = true
	if installer.TargetDir() != "/global" {
		t.Errorf("TargetDir() with UseGlobal = %q, want /global", installer.TargetDir())
	}
}
