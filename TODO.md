

# Chat UX
- When submitting the user text. It immediately needs to be visible in the chat history. Currently, it wont show until the request is finished
- Our timeout default needs to be longer. 90s and after 15s a small dim white text should populate the chat saying "processing...", after 30s "taking forever...", after "60s", "you're still here?"

# Settings mgmt
- We need to keep track of a `settings.example.yaml` that contains all of the avail configuration options. We will keep this up to date as we add more, however every field / configuration should have a default.
- The `cmd/root.go` file where viper is initialized needs to set defaults for all avail configurations. It might be best to move this into it's own package to keep it organized.
- passing cobra-cli args need to be bound to viper configs. any viper config should be able to be added as a namespaced cli argument. example: from yaml config ollama: url can be passed as `--ollama.url` and the same for all other arguments.
- Full review of config usage. Make sure that the configs are being used everywhere they should

# Log handling
- debug logging does not need to report every keypress. only core events and errors
- the debug log file name needs to be in the format <NAME>-<YYYYMMDD>_<HH:MM>.log
- the debug log file needs to respect the persistence config option

# model management view
- Better visual indication is need for when a model is selected. The blue highlight obfuscates that the model name color has changed
- When the footer status has an update: "Model changed" or something, it needs to remove the message after 2 seconds back to the default state like a toast message
- A simple progress bar integrated into the download modal. It will replace the input field and show a percentage by polling ollama for updates. If the user presses <esc> to leave the view, small text in the footer area should reflext that <model> is being pulled by putting a dim white text justified right saying and showing the spinner "<SPINNER> pulling: <MODEL_NAME>..."
- The model management view should refresh periodicly based on the config `ollama.poll_interval`


# BUGS
- Error messages are reported in 2 locations. Lets only use the one at the bottom of the chat.

# Duplicate or unused logic
- pkg/chat/conversation.go has isEmpty, GetMessagesAfter, Getmessagesbefore which seems to only be used in