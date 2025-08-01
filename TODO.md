~~1. The model management view needs padding all around as well as a skinny solid border~~
~~2. The j/k keys should move the selected item up and down.~~
~~3. The current selected model should be highlighted in the model management page. Only the text should be light yellow.~~
~~4. Pressing <enter> on a model changes the selected model and actually writes this change to the settings yaml~~
~~4. Pressing <esc> on any view other than chat should re-focus the chat view~~
~~5. The view menu should also have vim sytle j/k movement.~~
~~6. We need to make sure to handle the case where ollama is not avail and show a modal that says, "Cannot connect to ollama on <configured url>"~~
~~7. When we do have a connection to ollama, we need to make sure there is a model to use. If there is only one avail in ollama we need to make sure that the text showing the default model is red-strikethrough to show unavailability. This should be the case whenever any model that is set is not avail in ollama.~~
~~7. When we do have a connection to ollama, we need to make sure there is a model to use. If there is only one avail in ollama we need to make sure that the text showing the default model is red-strikethrough to show unavailability. This should be the case whenever any model that is set is not avail in ollama.~~
~~8. Refactor the "view select modal" to be more of a command pallette - a little wider, ho header, a straight simple list of views and actions.~~
~~- Pressing <ctrl-d> should prompt the user to delete the model with a confirmation modal~~
~~- The delete confirm modal needs to be simplified to read "Delete: <MODEL_NAME>\n Press <enter> to confirm, <esc> to cancel"~~
~~- Remove "Model:" text from the status footer. Only show the model name. Adjust the font color to be dim white~~
- There needs to be a lot more margin or padding around the whole TUI. The inside chat area should have slightly more margin / padding than the rest.
- The footer status bar needs to be simplified on the model management page. It should say: models: <NUM_MODELS> | size: <TOTAL_SIZE>
- The model view should refresh periodicly based on the config `ollama.poll_interval`
- in `.ryan/settings.yaml` the `ollama.system_prompt` field is now the path to a file to be read. If the file does not exist, fallback to the default.
- Configuration management with viper needs to be moved into its own package and defaults set for all things.
~~- On the model management view, add a keyboard command, "n" to bring up a modal text input field for entering in a model name to pull. Pressing <enter> will begin pulling this model.~~
- When the footer status has an update: "Model changed" or something, it needs to remove the message after 2 seconds back to the default state.
- From the model management view, help text at the bottom should show that pressing "n" will pull a new model and ctrl-d will delete.
- A simple progress bar integrated into the download modal. It will replace the input field and show a percentage by polling ollama for updates. If the user presses <esc> to leave the view, small text in the footer area should reflext that <model> is being pulled by putting a dim white text justified right saying and showing the spinner "<SPINNER> pulling: <MODEL_NAME>..."

# BUGS
- Error messages are reported in 2 locations. Lets only use the one at the bottom of the chat.
~~-  Look at the logs .ryan/debug.spinner-disapear.log. It appears that an error occured but there was no alert or text notifying one occurred appeared. This error text should be displayed as a red chat message.~~
