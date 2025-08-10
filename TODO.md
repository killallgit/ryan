



- ollama in tests needs to be always set via env var
- Each domain should handle setting its own config defaults
- We should have one config initialization package and inject the configs into functions for better composability. This will require updating all the tests to use this config injection as well
- sessionId's should be generated and not static.
- lets start organizing our providers better. They should be in their own package. pkg/providers/ollama is the only one right now. We can also take this time to do a review of how the provider is created, configured, and injected and whether there's obvious areas of improvement in the context of the app as a whole
