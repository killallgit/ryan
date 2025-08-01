~~1. The model management view needs padding all around as well as a skinny solid border~~
~~2. The j/k keys should move the selected item up and down.~~
~~3. The current selected model should be highlighted in the model management page. Only the text should be light yellow.~~
~~4. Pressing <enter> on a model changes the selected model and actually writes this change to the settings yaml~~
~~4. Pressing <esc> on any view other than chat should re-focus the chat view~~
~~5. The view menu should also have vim sytle j/k movement.~~
~~6. We need to make sure to handle the case where ollama is not avail and show a modal that says, "Cannot connect to ollama on <configured url>"~~
~~7. When we do have a connection to ollama, we need to make sure there is a model to use. If there is only one avail in ollama we need to make sure that the text showing the default model is red-strikethrough to show unavailability. This should be the case whenever any model that is set is not avail in ollama.~~
6. We need to make sure to handle the case where ollama is not avail and show a modal that says, "Cannot connect to ollama on <configured url>"
7. When we do have a connection to ollama, we need to make sure there is a model to use. If there is only one avail in ollama we need to make sure that the text showing the default model is red-strikethrough to show unavailability. This should be the case whenever any model that is set is not avail in ollama.
8. Refactor the "view select modal" to be more of a command pallette - a little wider, ho header, a straight simple list of views and actions.
9. On the model management view, add a keyboard command, "n" to bring up a modal text input field for entering in a model name to pull. Pressing <enter> will begin pulling this model.
10. On the modal management page, pressing ctrl-d should delete a model with a simple confirm prompt.
11. From the model management view, help text at the bottom should show that pressing "n" will pull a new model.
12. A simple progress bar integrated into the modal. It will replace the input field and show a percentage. If the user presses <esc> to leave the view, small text in the footer area should reflext that <model> is being pulled.