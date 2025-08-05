#!/bin/bash

echo "Running ryan with debug logging enabled..."
echo ""
echo "Test these scenarios:"
echo "1. Type 'hello' and press Enter"
echo "   - You should see activity: 'Assistant › generating response ●'"
echo ""
echo "2. Type 'ls -la' and press Enter"
echo "   - You should see activity with tool execution"
echo ""
echo "3. Check that you can send multiple messages"
echo ""
echo "Starting with debug logging..."
echo ""

# Enable debug logging to see all activity updates
export LOG_LEVEL=debug

# Run ryan
./ryan 2>&1 | grep -E "(activity|Activity|tree|Tree|Sent|Updated)" &
RYAN_PID=$!

# Also run ryan in another terminal for interaction
./ryan

# Clean up
kill $RYAN_PID 2>/dev/null
