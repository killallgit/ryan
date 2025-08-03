# UI/UX
- Our "status row" or the row above the chat input that holds the tokens and spinner should be refactored into this style: <SPINNER> <FEEDBACK_TEXT> (<DURATION> | <NUM_TOKENS> | <bold>esc</bold> to interject)
- BUG: Token count is not properly reported
- Buttons for download modal currently overflow the container. they need to be inside the solid line border and both buttons should fill the avail space. The inside of the buttons needs less padding around the text as well
- The delete model modal needs to use the same buttons as the other modals that have buttons.