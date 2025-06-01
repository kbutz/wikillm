#!/bin/bash

# Test script for the enhanced TODO list tool
# This demonstrates how the LLM can now analyze tasks beyond simple listing

echo "=== Enhanced TODO List Tool Test Cases ==="
echo ""

# Build the tool first
echo "Building the todo-agent..."
cd /Users/kyle.butz/go/src/github.com/kbutz/wikillm/tool
go build -o todo-agent . || exit 1
echo "Build successful!"
echo ""

# Test Case 1: Most important task query
echo "Test 1: Asking for the most important task"
echo "Query: 'What is my most important task today?'"
echo "Expected: Should use 'analyze priority' command to identify critical/high priority tasks"
echo ""

# Test Case 2: Task summary
echo "Test 2: Asking for a task summary"  
echo "Query: 'Can you give me a summary of all my tasks?'"
echo "Expected: Should use 'analyze summary' command to provide comprehensive overview"
echo ""

# Test Case 3: Time analysis
echo "Test 3: Asking about quick tasks"
echo "Query: 'What tasks can I complete quickly?'"
echo "Expected: Should use 'analyze time' command to identify quick wins"
echo ""

# Test Case 4: Export for complex analysis
echo "Test 4: Complex analytical query"
echo "Query: 'Which tasks should I prioritize based on importance and time required?'"
echo "Expected: Should use 'export' command to get full data for analysis"
echo ""

echo "=== Usage Instructions ==="
echo "1. Run: ./todo-agent"
echo "2. Try the test queries above"
echo "3. The tool should now properly analyze tasks instead of just listing them"
echo ""
echo "Example expected responses:"
echo "- For 'most important task': Will identify 'name the biggest slug Edward' as Critical priority"
echo "- For 'summary': Will show task counts, priorities, and recent additions"
echo "- For 'quick tasks': Will analyze time estimates (currently none set)"
echo ""
