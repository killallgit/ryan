# Chat input and footer info
- The footer status bar needs to be simplified on the model management page. It should say: models: <NUM_MODELS> | size: <TOTAL_SIZE>
- The processing spinner does not need extra text.
- Introduce the https://github.com/briandowns/spinner?tab=readme-ov-file spinner library and use spinner number 25
- token usage tracking should be in a format thats a little less intrusive and perhaps the footer should be organized with the "Ready" text as dim green justified to the left, the model name dim white justified all the way to the right and lets add the token display just above the chat input justified to the right. It can live with the same container that our spinner inhabits. That row should be spinner left justified, token readout in the form "<IN>/<OUT>" in dim white. this should also have the same padding as the input so it aligns exactly to the left and right of the input.

# Token calculation and display
- Token count should be a total of both up and down. 

# Settings mgmt
- in `.ryan/settings.yaml` the `ollama.system_prompt` field is now the path to a file to be read. If the file does not exist, fallback to the default.
- The root cmd file where viper is initialized needs to set defaults for all avail configurations. It might be best to move this into it's own package to keep it organized.

# model management view
- When the footer status has an update: "Model changed" or something, it needs to remove the message after 2 seconds back to the default state like a toast message
- A simple progress bar integrated into the download modal. It will replace the input field and show a percentage by polling ollama for updates. If the user presses <esc> to leave the view, small text in the footer area should reflext that <model> is being pulled by putting a dim white text justified right saying and showing the spinner "<SPINNER> pulling: <MODEL_NAME>..."
- The model management view should refresh periodicly based on the config `ollama.poll_interval`

# BUGS
- Error messages are reported in 2 locations. Lets only use the one at the bottom of the chat.

# Duplicate or unused logic
- pkg/chat/conversation.go has isEmpty, GetMessagesAfter, Getmessagesbefore which seems to only be used in