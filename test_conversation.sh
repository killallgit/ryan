#!/bin/bash

# Clear log for fresh test
> .ryan/system.log

# Start ryan in background
./bin/ryan 2>&1 &
RYAN_PID=$!

# Wait for startup
sleep 2

# Send first message (tool-based)
echo -e "how many files in this dir\n" | nc localhost 12345 2>/dev/null || true
sleep 3

# Send second message (conversational)
echo -e "what is the capital of France\n" | nc localhost 12345 2>/dev/null || true
sleep 3

# Kill the process
kill $RYAN_PID 2>/dev/null

# Show relevant logs
echo "=== DEBUG LOGS ==="
grep -E "(StartStreaming|Final response|MessageComplete|conversational|tool|sending|streaming)" .ryan/system.log | tail -50
