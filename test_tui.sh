#\!/bin/bash
# Test script to send input to TUI app

# Start ryan in background
./bin/ryan 2>&1 > ryan_output.log &
RYAN_PID=$\!

# Give it time to start
sleep 2

# Send "hello" followed by Enter using expect or similar
# Since we don't have expect, we'll use a different approach
# Kill the app after a delay
sleep 5
kill $RYAN_PID 2>/dev/null

# Check the logs
echo "=== Checking logs for our debug messages ==="
tail -100 .ryan/system.log | grep -E "(StartStreaming|SetSending|spinner|Message complete|processStreaming|send_message|chat_view)" || echo "No matching logs found"

echo "=== Last 20 log entries ==="
tail -20 .ryan/system.log
