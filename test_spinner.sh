#!/bin/bash

# Test script to verify spinner and error display functionality

echo "Testing the spinner and error display..."
echo "1. Run the application: ./ryan"
echo "2. Type a message and press Enter"
echo "3. You should see a spinner above the input field saying 'Sending message...'"
echo "4. If there's an error, it should appear in red in the same area"
echo ""
echo "To test error display:"
echo "1. Stop the ollama service: ollama stop"
echo "2. Send a message in the app"
echo "3. You should see an error message in red above the input field"