// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omniskill/skill"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got %d skills", r.Count())
	}
}

func TestRegister(t *testing.T) {
	r := New()

	s := &skill.BaseSkill{
		SkillName:        "test",
		SkillDescription: "Test skill",
	}

	if err := r.Register(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("expected 1 skill, got %d", r.Count())
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := New()

	s1 := &skill.BaseSkill{SkillName: "test"}
	s2 := &skill.BaseSkill{SkillName: "test"}

	if err := r.Register(s1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := r.Register(s2)
	if !errors.Is(err, ErrSkillExists) {
		t.Errorf("expected ErrSkillExists, got %v", err)
	}
}

func TestUnregister(t *testing.T) {
	r := New()

	s := &skill.BaseSkill{SkillName: "test"}
	_ = r.Register(s)

	if err := r.Unregister("test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Count() != 0 {
		t.Errorf("expected 0 skills, got %d", r.Count())
	}
}

func TestUnregisterNotFound(t *testing.T) {
	r := New()

	err := r.Unregister("nonexistent")
	if !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("expected ErrSkillNotFound, got %v", err)
	}
}

func TestGet(t *testing.T) {
	r := New()

	s := &skill.BaseSkill{
		SkillName:        "test",
		SkillDescription: "Test skill",
	}
	_ = r.Register(s)

	got, err := r.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Name() != "test" {
		t.Errorf("expected name 'test', got %q", got.Name())
	}
}

func TestGetNotFound(t *testing.T) {
	r := New()

	_, err := r.Get("nonexistent")
	if !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("expected ErrSkillNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	r := New()

	_ = r.Register(&skill.BaseSkill{SkillName: "a"})
	_ = r.Register(&skill.BaseSkill{SkillName: "b"})
	_ = r.Register(&skill.BaseSkill{SkillName: "c"})

	skills := r.List()
	if len(skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(skills))
	}
}

func TestListTools(t *testing.T) {
	r := New()

	tool1 := skill.NewTool("tool1", "Tool 1", nil, func(ctx context.Context, params map[string]any) (any, error) {
		return nil, nil
	})
	tool2 := skill.NewTool("tool2", "Tool 2", nil, func(ctx context.Context, params map[string]any) (any, error) {
		return nil, nil
	})

	s1 := &skill.BaseSkill{
		SkillName:  "skill1",
		SkillTools: []skill.Tool{tool1},
	}
	s2 := &skill.BaseSkill{
		SkillName:  "skill2",
		SkillTools: []skill.Tool{tool2},
	}

	_ = r.Register(s1)
	_ = r.Register(s2)

	tools := r.ListTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestGetTool(t *testing.T) {
	r := New()

	tool := skill.NewTool("greet", "Greet user", nil, func(ctx context.Context, params map[string]any) (any, error) {
		return "hello", nil
	})

	s := &skill.BaseSkill{
		SkillName:  "greeter",
		SkillTools: []skill.Tool{tool},
	}
	_ = r.Register(s)

	// Get by full name
	got, err := r.GetTool("greeter.greet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "greet" {
		t.Errorf("expected tool name 'greet', got %q", got.Name())
	}

	// Get by short name (searches all skills)
	got, err = r.GetTool("greet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "greet" {
		t.Errorf("expected tool name 'greet', got %q", got.Name())
	}
}

func TestGetToolNotFound(t *testing.T) {
	r := New()

	_, err := r.GetTool("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}

	_, err = r.GetTool("skill.nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill.tool")
	}
}

type mockSkill struct {
	skill.BaseSkill
	initCalled  bool
	closeCalled bool
	initErr     error
	closeErr    error
}

func (s *mockSkill) Init(ctx context.Context) error {
	s.initCalled = true
	return s.initErr
}

func (s *mockSkill) Close() error {
	s.closeCalled = true
	return s.closeErr
}

func TestInit(t *testing.T) {
	r := New()

	s1 := &mockSkill{BaseSkill: skill.BaseSkill{SkillName: "s1"}}
	s2 := &mockSkill{BaseSkill: skill.BaseSkill{SkillName: "s2"}}

	_ = r.Register(s1)
	_ = r.Register(s2)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s1.initCalled {
		t.Error("s1.Init was not called")
	}
	if !s2.initCalled {
		t.Error("s2.Init was not called")
	}
}

func TestInitError(t *testing.T) {
	r := New()

	expectedErr := errors.New("init error")
	s := &mockSkill{
		BaseSkill: skill.BaseSkill{SkillName: "failing"},
		initErr:   expectedErr,
	}
	_ = r.Register(s)

	err := r.Init(context.Background())
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestClose(t *testing.T) {
	r := New()

	s1 := &mockSkill{BaseSkill: skill.BaseSkill{SkillName: "s1"}}
	s2 := &mockSkill{BaseSkill: skill.BaseSkill{SkillName: "s2"}}

	_ = r.Register(s1)
	_ = r.Register(s2)

	if err := r.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s1.closeCalled {
		t.Error("s1.Close was not called")
	}
	if !s2.closeCalled {
		t.Error("s2.Close was not called")
	}
}

func TestCloseError(t *testing.T) {
	r := New()

	s := &mockSkill{
		BaseSkill: skill.BaseSkill{SkillName: "failing"},
		closeErr:  errors.New("close error"),
	}
	_ = r.Register(s)

	err := r.Close()
	if err == nil {
		t.Error("expected error from Close")
	}
}

func TestParseToolName(t *testing.T) {
	tests := []struct {
		input     string
		wantSkill string
		wantTool  string
	}{
		{"skill.tool", "skill", "tool"},
		{"my_skill.my_tool", "my_skill", "my_tool"},
		{"tool", "", "tool"},
		{"a.b.c", "a", "b.c"},
	}

	for _, tt := range tests {
		skillName, toolName := parseToolName(tt.input)
		if skillName != tt.wantSkill || toolName != tt.wantTool {
			t.Errorf("parseToolName(%q) = (%q, %q), want (%q, %q)",
				tt.input, skillName, toolName, tt.wantSkill, tt.wantTool)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := New()

	// Register skills concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			s := &skill.BaseSkill{SkillName: string(rune('a' + n))}
			_ = r.Register(s)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have some skills registered (exact count depends on race outcomes)
	if r.Count() == 0 {
		t.Error("expected some skills to be registered")
	}
}
