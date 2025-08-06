# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Refactored root initialization to improve modularity and separation of concerns
- Moved application logic from `cmd/root.go` into separate modules (`cmd/app.go`, `cmd/adapter.go`)
- Created initialization helpers in `pkg/agents/init.go` and `pkg/controllers/init.go`
- Improved error handling and logging during initialization
- **BREAKING**: Refactored model configuration to be provider-specific
  - Model is now configured per provider (`ollama.model`, `openai.model`)
  - Added `provider` field to select active LLM provider
  - CLI flags changed to provider-specific: `--ollama.model`, `--openai.model`

### Added
- Support for multiple LLM providers in configuration structure
- OpenAI configuration structure for future implementation
- Helper methods for accessing active provider configuration
- Provider selection via `--provider` CLI flag

### Removed
- Removed ScrumMaster agent and related tests (functionality can be achieved through orchestrator)
- Removed generic `--model` flag (replaced with provider-specific flags)

### Fixed
- Improved pre-commit hook compliance with proper file formatting
