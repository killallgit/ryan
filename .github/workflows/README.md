# GitHub Actions Workflows

## Integration Tests (`integration.yaml`)

The integration test workflow follows industry standard CI/CD practices:

### Trigger Events

1. **Pull Requests** - Runs on all PRs to main branch
   - Uses lightweight model (`smollm2:135m`) for fast feedback
   - 5-minute timeout

2. **Push to Main** - Runs when code is merged
   - Triggered by changes to `pkg/`, `cmd/`, `integration/`, or dependencies
   - Uses lightweight model for quick validation
   - 5-minute timeout

3. **Nightly Schedule** - Comprehensive testing at 2 AM UTC
   - Uses larger model (`qwen2.5:0.5b`) for thorough testing
   - 10-minute timeout for more extensive test coverage

4. **Manual Dispatch** - For debugging and custom testing
   - Choose from multiple Ollama models
   - Optional verbose output
   - Useful for reproducing issues

### Model Selection Strategy

- **PRs & Pushes**: `smollm2:135m` (fast, lightweight)
- **Nightly**: `qwen2.5:0.5b` (more comprehensive)
- **Manual**: User-selectable from predefined options

### Features

- Automatic model selection based on trigger type
- Configurable timeouts based on test scenario
- Test result summaries in GitHub UI
- Artifact upload for debugging failures
- Health checks for Ollama service

## Other Workflows

- **checks.yaml** - Linting and unit tests
- **coverage.yaml** - Code coverage reporting
- **release.yml** - Automated releases
- **claude.yml** - Claude AI integrations
- **claude-code-review.yml** - Automated code reviews
- **review.yaml** - PR review automation
