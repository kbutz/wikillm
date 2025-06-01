#!/bin/bash

# Test script to verify the loop fix in the enhanced TODO tool

echo "=== Testing Loop Fix for Enhanced TODO Tool ==="
echo ""

# Build the tool
echo "Building todo-agent..."
cd /Users/kyle.butz/go/src/github.com/kbutz/wikillm/tool
go build -o todo-agent . || exit 1
echo "Build successful!"
echo ""

echo "=== Test Instructions ==="
echo "1. Run: ./todo-agent"
echo "2. Test the following queries:"
echo ""
echo "Test Query 1: 'Can you summarize my tasks?'"
echo "Expected: A clean, single response with task summary"
echo ""
echo "Test Query 2: 'What is my most important task?'"
echo "Expected: Should identify 'name the biggest slug Edward' as Critical priority"
echo ""
echo "Test Query 3: 'Show me tasks I can do quickly'"
echo "Expected: Time-based analysis (note: current tasks don't have time estimates)"
echo ""
echo "=== What Was Fixed ==="
echo "1. Improved follow-up prompt with explicit instructions to avoid verbose output"
echo "2. Added cleanupResponse() function to remove duplicate content and thinking process"
echo "3. Enhanced prompt to start responses immediately without meta-commentary"
echo ""
echo "The agent should now provide clean, direct responses without loops or duplicates."
