#!/bin/bash

# Test script for improved TODO agent

echo "Building the improved TODO agent..."
go build -o todo-agent-improved .

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Starting the improved agent..."
echo ""
echo "Test queries to try:"
echo "1. What is my most important task today?"
echo "2. Can you show me my TODO list ranked by difficulty, showing my easiest tasks first?"
echo "3. Give me a summary of my tasks"
echo "4. Show me all my tasks by priority"
echo "5. What tasks do I have?"
echo ""

./todo-agent-improved -model default -provider lmstudio
