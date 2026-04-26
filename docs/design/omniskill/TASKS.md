# Tasks: OmniSkill Expansion

## Phase 1: Foundation

### Repository Restructure

- [x] **TASK-100**: Rename mcpkit → omniskill
  - Repository renamed
  - Git remote updated

- [x] **TASK-101**: Restructure package layout
  - Move `runtime/` → `mcp/server/`
  - Move `client/` → `mcp/client/`
  - Move `oauth2/` → `mcp/oauth2/`
  - Update internal imports

- [x] **TASK-102**: Update module path
  - Change `github.com/plexusone/mcpkit` → `github.com/plexusone/omniskill`
  - Update go.mod
  - Update all import statements

- [ ] **TASK-103**: Backwards compatibility
  - Add module aliases at old paths
  - Document migration guide
  - Update CHANGELOG

### Core Interfaces

- [ ] **TASK-110**: Create `skill/` package
  - Define Skill interface
  - Define Tool interface
  - Define Parameter type
  - Add comprehensive tests

- [ ] **TASK-111**: Create skill builder
  - Fluent API for constructing skills
  - Validation on build
  - Tests

### Registry

- [ ] **TASK-120**: Create `registry/` package
  - Define Registry interface
  - In-memory implementation
  - Thread-safe operations
  - Tests

---

## Phase 2: MCP Integration

### Server Integration

- [ ] **TASK-200**: MCP server skill support
  - Add `RegisterSkill()` to Runtime
  - Convert skill.Tool → mcp.Tool automatically
  - Tests

- [ ] **TASK-201**: Auto-registration option
  - Add `WithAutoRegister()` option
  - Register skills with omniskill registry on server start
  - Tests

### Client Integration

- [ ] **TASK-210**: MCP session as skill
  - Add `Session.AsSkill()` method
  - Wrap mcp.Tool as skill.Tool
  - Cache discovered tools
  - Tests

---

## Phase 3: OpenAPI Import

### Parser

- [ ] **TASK-300**: OpenAPI parser
  - Parse OpenAPI 3.0 and 3.1 specs
  - Support JSON and YAML formats
  - Extract operations as tools

- [ ] **TASK-301**: Schema conversion
  - Convert OpenAPI schemas to Parameter types
  - Handle refs and nested schemas
  - Support common formats (date, email, uuid)

### Skill Generator

- [ ] **TASK-310**: OpenAPI skill generator
  - Generate skill from parsed spec
  - HTTP client with configurable base URL
  - Request/response handling

- [ ] **TASK-311**: Authentication support
  - API key authentication
  - Bearer token authentication
  - OAuth2 flows (optional)

---

## Phase 4: Export Formats

### Compiled Export

- [ ] **TASK-400**: Export to compiled.Skill
  - Convert skill.Skill → omniagent compiled.Skill
  - Zero-overhead for Go consumers
  - Integration test with omniagent

### OpenClaw Export

- [ ] **TASK-410**: Research OpenClaw format
  - Document specification
  - Identify mapping to skill.Skill

- [ ] **TASK-411**: OpenClaw exporter
  - Convert skill.Skill → OpenClaw format
  - Tests with sample skills

### Claude Code Export

- [ ] **TASK-420**: Research Claude Code format
  - Document tool specification
  - Identify mapping to skill.Skill

- [ ] **TASK-421**: Claude Code exporter
  - Convert skill.Skill → Claude Code format
  - Tests

---

## Current Progress

**Last mcpkit Release**: v0.5.0 (2026-04-26) - Added MCP client package

**Current Phase**: Phase 1 - Foundation

### Next Actions

1. ~~Complete TASK-101: Restructure package layout~~ ✅
2. ~~Complete TASK-102: Update module path~~ ✅
3. Add backwards compatibility (TASK-103)
4. Create skill/ and registry/ packages (TASK-110, TASK-120)

### Dependencies

| Downstream | Status | Notes |
|------------|--------|-------|
| omniagent | Pending | Must update imports after TASK-102 |
