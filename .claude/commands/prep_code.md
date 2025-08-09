# Prepare code for integration

- Claude will run all tests, including integration tests.
- Claude makes sure the introduced code has proper code coverage
- Claude will identify any external env vars or settings that might need to be added to example files
- Claude will do a code review of changes paying special attention to potentially duplicated logic
- Claude will create a branch (if not on a working branch) and create a draft PR
- Claude will update the changelog and related readmes
- When checks are passing, Claude will push the feature / fix branch
