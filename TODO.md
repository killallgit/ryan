# Introduce the idea of "modes"
- There should be operating modes so that the user can decide whether or not a request requires thinking or tool usage. The first step is to integrate the "chatMode" idea into the app in the most basic form so that its able to adjust how prompting is processed. This first step will not change any of the current prompting and message receiving yet
- Part 2 is to design the "modes" These modes should adjust the system prompt and other key components to create separate "personalities". Each personality system prompt and other instructions should be defined in their own markdown file
# Support for more than just agent models
- We need to create an idea of "modes" modes 

# Log handling
- debug logging does not need to report every keypress. only core events and errors
- the debug log file name needs to be in the format <NAME>-<YYYYMMDD>_<HH:MM>.log
- the debug log file needs to respect the persistence config option

# model management view
- The model management view should refresh periodicly based on the config `ollama.poll_interval`

# Duplicate or unused logic
- pkg/chat/conversation.go has isEmpty, GetMessagesAfter, Getmessagesbefore which seems to only be used in