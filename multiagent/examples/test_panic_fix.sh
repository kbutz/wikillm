#!/bin/bash

# Quick test to verify panic fix
echo "🔧 Testing Panic Fix"
echo "===================="

cd "$(dirname "$0")"

# Clean memory
rm -rf wikillm_memory/memory/*

# Build
echo "Building..."
go build -o interactive_example interactive_example.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "🚀 Testing with communication query..."
LOGFILE="panic_test.log"

(
    echo "help me draft a professional email"
    sleep 30  # Give time for coordination
    echo "exit"
) | timeout 45s ./interactive_example 2>&1 | tee $LOGFILE

echo ""
echo "📊 Results:"

# Check for panic
if grep -q "panic:" $LOGFILE; then
    echo "❌ PANIC STILL OCCURS!"
    grep -A 5 "panic:" $LOGFILE
    exit 1
else
    echo "✅ No panic detected"
fi

# Check if coordination completed
if grep -q "CoordinatorAgent: LLM synthesis completed" $LOGFILE; then
    echo "✅ LLM synthesis completed"
else
    echo "ℹ️  LLM synthesis not reached"
fi

if grep -q "CoordinatorAgent: Successfully sent final response" $LOGFILE; then
    echo "✅ Final response sent successfully"
else
    echo "❌ Final response not sent"
fi

if grep -q "Personal Assistant:" $LOGFILE; then
    echo "✅ User received response"
else
    echo "❌ User did not receive response"
fi

# Count messages to check for reasonable flow
TOTAL_MESSAGES=$(grep -c "Routing message" $LOGFILE 2>/dev/null || echo 0)
echo "📈 Total messages: $TOTAL_MESSAGES"

if [ $TOTAL_MESSAGES -gt 0 ] && [ $TOTAL_MESSAGES -lt 50 ]; then
    echo "✅ Normal message flow"
else
    echo "⚠️  Unusual message count"
fi

echo ""
if grep -q "Personal Assistant:" $LOGFILE && ! grep -q "panic:" $LOGFILE; then
    echo "🎉 SUCCESS: Panic fixed and system working!"
    exit 0
else
    echo "❌ Issues remain - check log: $LOGFILE"
    exit 1
fi
