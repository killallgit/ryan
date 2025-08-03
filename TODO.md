
# Context
- store context to disk per session.


# Integrate langchain.
- context management

# Log handling
- debug logging does not need to report every keypress. only core events and errors
- the debug log file name needs to be in the format <NAME>-<YYYYMMDD>_<HH:MM>.log
- the debug log file needs to respect the persistence config option

# model management view
- The model management view should refresh periodicly based on the config `ollama.poll_interval`

# Duplicate or unused logic
- pkg/chat/conversation.go has isEmpty, GetMessagesAfter, Getmessagesbefore which seems to only be used in