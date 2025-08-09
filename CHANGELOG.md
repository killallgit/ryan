# Changelog

## [Unreleased]

### Added
- Status bar process icons in TUI to show current system state (↑ sending, ↓ receiving, 🤔 thinking, 🔨 tool usage)
- Shared process state constants package (`pkg/process`) for consistent state management across the application
- Unit tests for process state package with 100% coverage

### Changed
- Renamed all references to "orchestrator" to the more generic term "agent" throughout the codebase for better clarity and consistency
- Status bar now displays dynamic icons and states during message processing
