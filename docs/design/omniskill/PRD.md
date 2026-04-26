# Product Requirements Document: OmniSkill

## Overview

OmniSkill is a unified skill/tool infrastructure library for Go that enables defining skills once and exposing them across multiple formats and protocols.

## Problem Statement

The AI tooling ecosystem is fragmented:
- MCP (Model Context Protocol) for Claude, ChatGPT
- OpenAPI for REST APIs
- OpenClaw for standardized AI tool definitions
- Claude Code for development tooling
- Custom compiled skills for Go applications (e.g., omniagent)

Developers must implement the same functionality multiple times for different platforms, leading to:
- Code duplication
- Inconsistent behavior across platforms
- Maintenance burden
- Slow adoption of new platforms

## Goals

1. **Define Once, Deploy Everywhere** - Write skill logic once in Go, expose via multiple formats
2. **MCP-First** - Full MCP protocol support (server + client) as the primary interop layer
3. **Skill Registry** - Centralized discovery and management of available skills
4. **Format Converters** - Import/export skills from OpenAPI, OpenClaw, Claude Code formats
5. **Go-Native Performance** - Zero-overhead for Go consumers (compiled skills)

## Non-Goals

- Runtime for non-Go languages (skills must be implemented in Go)
- Hosting/deployment platform (omniskill is a library)
- UI/dashboard for skill management

## User Personas

### Skill Developer
- Builds reusable tools/skills in Go
- Wants to expose skills to multiple AI platforms
- Needs clear APIs and documentation

### AI Application Developer
- Builds applications using AI (e.g., omniagent)
- Wants to consume skills from various sources
- Needs unified interface regardless of skill origin

### Platform Integrator
- Integrates AI tooling into existing systems
- Needs standard protocols (MCP, REST)
- Wants registry/discovery capabilities

## Requirements

### P0 - Must Have

| ID | Requirement | Rationale |
|----|-------------|-----------|
| R1 | MCP server implementation | Core protocol for AI interop |
| R2 | MCP client implementation | Connect to external MCP servers |
| R3 | Skill interface definition | Common abstraction for all skills |
| R4 | Tool interface definition | Atomic callable unit |
| R5 | Skill registry | Discovery and management |

### P1 - Should Have

| ID | Requirement | Rationale |
|----|-------------|-----------|
| R6 | OpenAPI importer | Generate skills from existing APIs |
| R7 | Auto-registration | MCP servers built with omniskill auto-register |
| R8 | compiled.Skill export | Direct Go consumption without MCP overhead |

### P2 - Nice to Have

| ID | Requirement | Rationale |
|----|-------------|-----------|
| R9 | OpenClaw compatibility | Broader ecosystem support |
| R10 | Claude Code format | Development tooling integration |
| R11 | Skill templates | Quick-start for common patterns |

## Success Metrics

1. **Adoption** - Number of skills built with omniskill
2. **Platform Coverage** - Number of export formats supported
3. **Developer Experience** - Time to build and deploy a skill
4. **Performance** - Overhead compared to native implementations

## Timeline

| Phase | Focus | Target |
|-------|-------|--------|
| Phase 1 | MCP server + client (from mcpkit) | v0.5.0 |
| Phase 2 | Skill/Tool interfaces, Registry | v0.6.0 |
| Phase 3 | OpenAPI importer | v0.7.0 |
| Phase 4 | OpenClaw, Claude Code | v0.8.0 |

## Open Questions

1. Should registry support remote discovery (network-based)?
2. How to handle authentication across different platforms?
3. Should skills support versioning/deprecation?
