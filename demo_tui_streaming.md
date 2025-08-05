# TUI Streaming Fix Demonstration

## Problem Description
When the TUI was outputting code during streaming responses, the code would "erase itself" - appearing briefly then disappearing as new content was streamed in. This made it impossible to read code blocks properly during streaming.

## Root Cause
The issue was in the rendering pipeline:
1. During each streaming update, `clearArea(screen, area)` was called on the entire message area
2. This cleared all previously rendered content
3. Then the entire message history + new streaming content was re-rendered
4. This caused a flicker effect where code would appear and disappear

## Solution
Implemented an incremental streaming renderer (`StreamingRenderer`) that:
1. Tracks what content has already been rendered
2. Only renders new content without clearing the entire area
3. Falls back to full render when necessary (e.g., when scrolling or starting new stream)

## Key Changes

### 1. Created `render_streaming.go`
- `StreamingRenderer` struct to track rendering state
- `RenderStreamingIncremental` method for incremental updates
- `RenderMessagesWithIncrementalStreaming` wrapper function

### 2. Updated `ChatView`
- Added `streamingRenderer` field
- Modified `Render()` to use incremental rendering during streaming
- Reset renderer state on new streams

### 3. Fixed Code Review Agent
- Now properly handles single file paths (not just directories)
- Returns the file directly instead of trying to search within it

## Testing

### CLI Mode (Working)
```bash
./ryan --prompt "review code in pkg/agents/orchestrator.go" --no-tui
```

### TUI Mode (Fixed)
1. Launch: `./ryan`
2. Type: `review code in pkg/agents`
3. Observe: Code blocks should remain visible during streaming without erasing

## Benefits
1. Smooth streaming experience in TUI
2. Code remains readable during output
3. Better performance (less re-rendering)
4. Maintains scroll position during streaming