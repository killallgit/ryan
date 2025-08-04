# UI/UX
- Integrate https://github.com/charmbracelet/glow for formatting markdown from LLM responses

- I want to give proper feedback that a tool is being called. Lets show feedback in the chat that looks like: <TOOL>(<truncated-command>). example. "Shell(docker ps -a)" or "Search("https://www.wikipedia.com")"

# Example desired terminal output:
```
⏺ Update Todos
  ⎿  ☐ Create stream_parser.go with StreamParser struct       
     ☐ Implement stateful parsing with buffer for partial tags
     ☐ Update HandleStreamUpdate to use StreamParser
     ☐ Create FormattedSegment type for rendering
     ☐ Update message rendering to handle formatted segments
     ☐ Test streaming with partial think tags
     ☐ Add support for nested formatting
     ☐ Prepare for future markdown support

✻ Thinking…

  Let me create the stream_parser.go file with the StreamParser struct and basic 
  types.

⏺ Write(../tui/stream_parser.go)
  ⎿  Wrote 268 lines to ../tui/stream_parser.go         
     package tui
     import (
       "strings"
       "github.com/gdamore/tcell/v2"
       "github.com/killallgit/ryan/pkg/logger"
     )
     // FormatType represents different formatting types
     … +258 lines (ctrl+r to expand)

✻ Thinking…

  Now let me create tests for the stream parser to ensure it handles partial tags 
  correctly.

⏺ Update Todos
  ⎿  ☒ Create stream_parser.go with StreamParser struct       
     ☒ Create FormattedSegment type for rendering
     ☐ Implement stateful parsing with buffer for partial tags
     ☐ Update HandleStreamUpdate to use StreamParser
     ☐ Update message rendering to handle formatted segments
     ☐ Test streaming with partial think tags
     ☐ Add support for nested formatting
     ☐ Prepare for future markdown support

✻ Thinking…

  Let me create test cases for the stream parser to ensure it properly handles 
  partial tags and various streaming scenarios.

✻ Shimmying… (102s · ⚒ 8.7k tokens · esc to interrupt)
```


