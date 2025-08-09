# name: Create a PR for the current modified code

## Description: Executes the steps required to create a standard PR

### Instructions
- Cleanup any example, demo, or one off test scripts that may be left around
- Run tests and linting with `task check` and fix any issues
- Make sure the CHANGELOG.md is up to date
- Make sure all related code has been commited and a proper commit message is given in the form of [<DOMAIN>]: <SHORT_DESCRIPTION>
- Pull origin main to rebase fix merge conflicts. Make sure rebase is completed. Re add and commit merged files with --amend
- Re-run all tests and checks if rebasing added new code then commit once all checks and tests are passing
- Create a PR using the gh cli. Fill out a detailed description
