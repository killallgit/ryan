#!/bin/bash
# Test script for TUI streaming output

echo "Testing TUI streaming with code output..."
echo ""
echo "This test will:"
echo "1. Launch the TUI"
echo "2. Ask for a code review to trigger streaming output with code"
echo "3. Observe if the code output erases itself during streaming"
echo ""
echo "Instructions:"
echo "1. When the TUI opens, type: 'review code in pkg/agents'"
echo "2. Watch the output carefully - code should NOT erase itself"
echo "3. Press Ctrl+C to exit when done"
echo ""
echo "Press Enter to start the test..."
read

# Run the application in TUI mode
./ryan