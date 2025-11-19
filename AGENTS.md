# AGENTS.md - AI Contributor Guide

## Purpose
This repository contains BlueBuild configuration and declarative configs to build Fedora CoreOS-based homelab images for NAB9 mini PCs. Use this guide as the authoritative reference for AI-driven changes.

## Scope
Applies to all files in this repository unless superseded by a more specific AGENTS.md.

## Development Principles
- Match existing patterns and naming; prefer small, focused commits.
- Protect correctness and stability: validate inputs, avoid shell injection, and keep paths safe.
- Keep documentation in sync with behavior changes.

## Tooling & Commands
- Primary tools: BlueBuild recipes, Butane/Ignition configs, systemd units, shell scripts.
- Images: BlueBuild recipes under `recipes/`; provisioning via Butane/Ignition in `ignition/`.
- Binary updates: GitHub Actions automatically fetches and builds the `homelab-setup` binary from the separate [plex-migration-homelab/homelab-setup](https://github.com/plex-migration-homelab/homelab-setup) repository.

## Repository Structure
- `recipes/`: BlueBuild image recipe and manifests.
- `files/`: System overlays, scripts, and bundled assets (including the compiled `homelab-setup` binary at `files/system/usr/bin/homelab-setup`).
- `ignition/`: Butane templates and transpile scripts.
- `docs/`: Guides, references, and testing checklists.
- `modules/`: Custom BlueBuild modules.
- `.github/workflows/`: GitHub Actions for building the homelab-setup binary from upstream.

## Git Workflow
- Feature branches should use `claude/<description>-<session-id>` when creating new branches.
- Commit messages: `type: summary`, e.g., `chore: add agent guide`.
- Run tests/lints relevant to your changes before committing.

## Expectations for AI Contributions
- Keep changes minimal, well-reasoned, and fully documented.
- Favor readability over cleverness; avoid unnecessary abstractions.
- Never wrap imports in try/catch (not applicable for Go but applies to other languages).
- When adding prompts or user-facing strings, ensure clarity and consistency.

## Security & Reliability
- Validate user input, ports, IPs, and file paths.
- Avoid hardcoding secrets; use configuration where necessary.
- Ensure systemd units, scripts, and configs remain idempotent and safe to rerun.

## Documentation Updates
- Update relevant docs in `docs/` or inline comments when behavior changes.
- Keep change logs accurate when modifying release-related files.

## Getting Help
- Refer to `CLAUDE.md` for deeper project context.
- Inspect similar files for patterns before introducing new ones.
