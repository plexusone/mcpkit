# Implementation Plan: OmniSkill

## Phase 1: Foundation (v0.5.0 → v0.6.0)

### Step 1.1: Restructure mcpkit → omniskill

**Status:** In Progress

Reorganize existing mcpkit code into omniskill structure:

```
mcpkit/                    →  omniskill/
├── runtime/               →  ├── mcp/server/
├── client/                →  ├── mcp/client/
├── oauth2/                →  └── mcp/oauth2/
```

Tasks:
- [x] Rename repository mcpkit → omniskill
- [x] Update git remote
- [ ] Move runtime/ → mcp/server/
- [ ] Move client/ → mcp/client/
- [ ] Move oauth2/ → mcp/oauth2/
- [ ] Update go.mod module path
- [ ] Update all internal imports
- [ ] Add module aliases for backwards compatibility

### Step 1.2: Define Core Interfaces

Create `skill/` package with core types:

- [ ] `skill/skill.go` - Skill interface
- [ ] `skill/tool.go` - Tool interface and Parameter type
- [ ] `skill/builder.go` - Fluent builder for creating skills
- [ ] `skill/skill_test.go` - Tests

### Step 1.3: Create Registry

Create `registry/` package:

- [ ] `registry/registry.go` - Registry interface and implementation
- [ ] `registry/memory.go` - In-memory registry
- [ ] `registry/registry_test.go` - Tests

## Phase 2: MCP Integration (v0.6.0)

### Step 2.1: Integrate Skills with MCP Server

Update `mcp/server/` to support skills:

- [ ] Add `RegisterSkill(skill.Skill)` method
- [ ] Auto-convert skill tools to MCP tools
- [ ] Add `WithAutoRegister()` option for registry integration
- [ ] Update tests

### Step 2.2: MCP Client as Skill

Update `mcp/client/` to expose sessions as skills:

- [ ] Add `Session.AsSkill(name string)` method
- [ ] Wrap MCP tools as skill.Tool
- [ ] Handle tool discovery and caching
- [ ] Update tests

## Phase 3: OpenAPI Import (v0.7.0)

### Step 3.1: OpenAPI Parser

Create `openapi/` package:

- [ ] `openapi/parser.go` - Parse OpenAPI 3.x specs
- [ ] `openapi/operation.go` - Convert operations to tools
- [ ] `openapi/schema.go` - Convert schemas to parameters
- [ ] `openapi/parser_test.go` - Tests with sample specs

### Step 3.2: OpenAPI Skill Generator

- [ ] `openapi/skill.go` - Generate skill from parsed spec
- [ ] `openapi/client.go` - HTTP client for API calls
- [ ] `openapi/auth.go` - Handle API authentication
- [ ] Integration tests

## Phase 4: Export Formats (v0.8.0)

### Step 4.1: Compiled Skill Export

Create `export/compiled/` package:

- [ ] `export/compiled/export.go` - Convert to omniagent compiled.Skill
- [ ] Tests with omniagent integration

### Step 4.2: OpenClaw Export

Create `export/openclaw/` package:

- [ ] Research OpenClaw specification
- [ ] `export/openclaw/export.go` - Convert to OpenClaw format
- [ ] Tests

### Step 4.3: Claude Code Export

Create `export/claudecode/` package:

- [ ] Research Claude Code tool format
- [ ] `export/claudecode/export.go` - Convert to Claude Code format
- [ ] Tests

## Verification Checkpoints

### v0.6.0 Release Criteria

1. All mcpkit functionality preserved under new paths
2. Core skill/tool interfaces defined and documented
3. Registry functional with in-memory implementation
4. MCP server can register skills
5. MCP client sessions can be used as skills
6. All existing tests pass
7. Backwards compatibility aliases work

### v0.7.0 Release Criteria

1. OpenAPI 3.x specs can be imported
2. Generated skills make correct HTTP calls
3. Authentication (API key, Bearer, OAuth) supported
4. Error handling for API failures

### v0.8.0 Release Criteria

1. Skills can be exported to omniagent compiled.Skill
2. Skills can be exported to OpenClaw format
3. Skills can be exported to Claude Code format
4. Round-trip tests pass (import → export → import)

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking changes for mcpkit users | Provide module aliases, deprecation period |
| OpenClaw spec unclear | Research early, reach out to maintainers |
| Performance regression | Benchmark before/after each phase |

## Dependencies

- omniagent must update to consume from omniskill instead of mcpkit
- omnistorage must be released for omniagent integration
