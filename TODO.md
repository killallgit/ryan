


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
- The model management view should refresh periodicly based on the config `ollama.poll_interval`


# BUGS
- Error messages are reported in 2 locations. Lets only use the one at the bottom of the chat.

# Duplicate or unused logic
- pkg/chat/conversation.go has isEmpty, GetMessagesAfter, Getmessagesbefore which seems to only be used in