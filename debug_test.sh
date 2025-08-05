#!/bin/bash

echo "Testing with debug output to verify spinner/activity is working..."
echo ""
echo "Watch for these in the debug output:"
echo "- 'Sending message' when you type something"
echo "- 'Stream started' when processing begins"
echo "- 'Sent initial activity update' showing the tree"
echo ""
echo "Starting ryan with debug logging to stderr..."
echo ""

LOG_LEVEL=debug ./ryan 2>&1 | tee ryan_debug.log &
PID=$!

echo ""
echo "Ryan is running. Try sending a message."
echo "Debug output is being saved to ryan_debug.log"
echo ""
echo "Press Ctrl+C to stop"

wait $PID
