#!/bin/bash

echo "=== Testing Unified Agent System ==="
echo ""

echo "1. Testing basic functionality (should work like before):"
echo "RYAN_OLLAMA_MODEL=test-model ./bin/ryan --headless --prompt 'What is 2+2?' --skip-permissions"
RYAN_OLLAMA_MODEL=test-model ./bin/ryan --headless --prompt "What is 2+2?" --skip-permissions
echo ""
echo ""

echo "2. Testing custom system prompt:"
echo "RYAN_OLLAMA_MODEL=test-model ./bin/ryan --system-prompt 'You are a math tutor.' --headless --prompt 'What is 2+2?' --skip-permissions"
RYAN_OLLAMA_MODEL=test-model ./bin/ryan --system-prompt "You are a math tutor." --headless --prompt "What is 2+2?" --skip-permissions
echo ""
echo ""

echo "3. Testing append system prompt:"
echo "RYAN_OLLAMA_MODEL=test-model ./bin/ryan --append-system-prompt 'Always explain your reasoning.' --headless --prompt 'What is 2+2?' --skip-permissions"
RYAN_OLLAMA_MODEL=test-model ./bin/ryan --append-system-prompt "Always explain your reasoning." --headless --prompt "What is 2+2?" --skip-permissions
echo ""
echo ""

echo "4. Testing planning bias flag:"
echo "RYAN_OLLAMA_MODEL=test-model ./bin/ryan --planning-bias --headless --prompt 'Help me refactor this code' --skip-permissions"
RYAN_OLLAMA_MODEL=test-model ./bin/ryan --planning-bias --headless --prompt "Help me refactor this code" --skip-permissions
echo ""
echo ""

echo "5. Testing combined flags:"
echo "RYAN_OLLAMA_MODEL=test-model ./bin/ryan --system-prompt 'You are a coding assistant.' --append-system-prompt 'Always be thorough.' --planning-bias --headless --prompt 'Write a function' --skip-permissions"
RYAN_OLLAMA_MODEL=test-model ./bin/ryan --system-prompt "You are a coding assistant." --append-system-prompt "Always be thorough." --planning-bias --headless --prompt "Write a function" --skip-permissions
echo ""
echo ""

echo "=== Testing Complete ==="
