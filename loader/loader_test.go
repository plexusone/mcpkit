// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/plexusone/omniskill/skill"
)

func TestInspect(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create skill directory with SKILL.md
	skillDir := filepath.Join(tmpDir, "weather")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: weather
description: "Weather forecasts"
---
# Weather Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600); err != nil {
		t.Fatal(err)
	}

	// Also create skill.go
	if err := os.WriteFile(filepath.Join(skillDir, "skill.go"), []byte("package weather"), 0600); err != nil {
		t.Fatal(err)
	}

	info, err := Inspect(skillDir)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if info.Name != "weather" {
		t.Errorf("Name = %q, want %q", info.Name, "weather")
	}

	if !info.HasMarkdown() {
		t.Error("HasMarkdown() = false, want true")
	}

	if !info.HasGo() {
		t.Error("HasGo() = false, want true")
	}

	if len(info.Formats) != 2 {
		t.Errorf("Formats length = %d, want 2", len(info.Formats))
	}
}

func TestInspect_NoSkill(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Inspect(tmpDir)
	if err == nil {
		t.Error("Expected error for empty directory")
	}
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skill directories
	for _, name := range []string{"weather", "github", "empty"} {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		// Only add SKILL.md to weather and github
		if name != "empty" {
			content := "---\nname: " + name + "\ndescription: \"" + name + " skill\"\n---\n# " + name
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}

	skills, err := DiscoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSkills failed: %v", err)
	}

	// Should find weather and github, not empty
	if len(skills) != 2 {
		t.Errorf("Found %d skills, want 2", len(skills))
	}
}

func TestGoSkillRegistry(t *testing.T) {
	reg := NewGoSkillRegistry()

	// Register a mock skill
	reg.Register("test", func() skill.Skill {
		return &skill.BaseSkill{
			SkillName:        "test",
			SkillDescription: "Test skill",
		}
	})

	if !reg.Has("test") {
		t.Error("Has(test) = false, want true")
	}

	if reg.Has("nonexistent") {
		t.Error("Has(nonexistent) = true, want false")
	}

	s, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if s.Name() != "test" {
		t.Errorf("Name = %q, want %q", s.Name(), "test")
	}

	names := reg.List()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("List = %v, want [test]", names)
	}
}

func TestUnifiedLoader(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill directory with SKILL.md
	skillDir := filepath.Join(tmpDir, "weather")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: weather
description: "Weather forecasts"
---
# Weather Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewUnifiedLoader()

	// Load without Go registration - should use SKILL.md
	s, format, err := loader.Load(skillDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if format != FormatMarkdown {
		t.Errorf("Format = %q, want %q", format, FormatMarkdown)
	}

	if s.Name() != "weather" {
		t.Errorf("Name = %q, want %q", s.Name(), "weather")
	}
}

func TestUnifiedLoader_PrefersGo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill directory with both formats
	skillDir := filepath.Join(tmpDir, "weather")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: weather
description: "Weather from SKILL.md"
---
# Weather
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "skill.go"), []byte("package weather"), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewUnifiedLoader()

	// Register Go implementation
	loader.RegisterGo("weather", func() skill.Skill {
		return &skill.BaseSkill{
			SkillName:        "weather",
			SkillDescription: "Weather from Go",
		}
	})

	s, format, err := loader.Load(skillDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should prefer Go
	if format != FormatGo {
		t.Errorf("Format = %q, want %q", format, FormatGo)
	}

	if s.Description() != "Weather from Go" {
		t.Errorf("Description = %q, want %q", s.Description(), "Weather from Go")
	}
}

func TestUnifiedLoader_LoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skill directories
	for _, name := range []string{"weather", "github"} {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		content := "---\nname: " + name + "\ndescription: \"" + name + " skill\"\n---\n# " + name
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	loader := NewUnifiedLoader()
	skills, err := loader.LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Loaded %d skills, want 2", len(skills))
	}
}

func TestSkillInfo_LoadMarkdown(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: test
description: "Test skill"
---
# Test
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600); err != nil {
		t.Fatal(err)
	}

	info, err := Inspect(skillDir)
	if err != nil {
		t.Fatal(err)
	}

	s, err := info.LoadMarkdown()
	if err != nil {
		t.Fatalf("LoadMarkdown failed: %v", err)
	}

	if s.Name() != "test" {
		t.Errorf("Name = %q, want %q", s.Name(), "test")
	}
}
