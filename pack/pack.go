// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package pack provides interfaces for markdown skill packs.
//
// A SkillPack bundles multiple markdown skills (SKILL.md files) into
// a single Go module using go:embed. Skill packs can be imported and
// used with any omniskill-compatible agent.
//
// Example implementation:
//
//	//go:embed skills/*
//	var skillsFS embed.FS
//
//	type Pack struct{}
//
//	func (Pack) Name() string    { return "my-skills" }
//	func (Pack) Version() string { return "d4eb236..." }
//	func (Pack) FS() embed.FS    { return skillsFS }
//
//	func Default() *Pack { return &Pack{} }
package pack

import "embed"

// SkillPack provides embedded markdown skills.
//
// Skill packs bundle multiple SKILL.md files following the OpenClaw format
// into a single Go module. This enables:
//   - Zero external dependencies at runtime
//   - Versioned skill bundles via Go modules
//   - Easy distribution and updates
//
// The FS() method should return an embedded filesystem with skills
// located at skills/<name>/SKILL.md.
type SkillPack interface {
	// Name returns the pack identifier (e.g., "omniagent-skills").
	Name() string

	// Version returns the pack version or source commit hash.
	// For packs derived from external sources (like OpenClaw),
	// this should be the source commit hash for traceability.
	Version() string

	// FS returns the embedded filesystem containing skills.
	// Skills are expected at skills/<name>/SKILL.md following
	// the OpenClaw SKILL.md format with YAML frontmatter.
	FS() embed.FS
}
