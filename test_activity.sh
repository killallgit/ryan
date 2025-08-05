#!/bin/bash

# Test script to verify activity indicator works

echo "Starting ryan with activity tracking test..."
echo ""
echo "Try these commands to test the activity indicator:"
echo "1. Ask it to list files: 'ls -la'"
echo "2. Ask it to search for something: 'find all go files in the current directory'"
echo "3. Ask it to read a file: 'show me the contents of main.go'"
echo ""
echo "You should see the activity tree above the input showing:"
echo "├── Assistant › generating response ●"
echo "└── ChatController › bash(ls -la) ●"
echo ""
echo "Starting ryan..."

./ryan
