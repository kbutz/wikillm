#!/bin/bash

# Test handler lifecycle and final delivery
echo "🔧 Testing Handler Lifecycle Fix"
echo "================================"

cd "$(dirname "$0")"

# Clean state
rm -rf wikillm_memory/memory/*

# Build
echo "Building..."
go build -o interactive_example interactive_example.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "🚀 Testing with extended timeout..."
LOGFILE="handler_test.log"

(
    echo "help me compose a professional email"
    sleep 45  # Wait for full coordination
    echo "exit"
) | timeout 180s ./interactive_example 2>&1 | tee $LOGFILE

echo ""
echo "📊 Handler Lifecycle Analysis:"
echo "=============================="

# Check handler registration
if grep -q "Registered user response handler for key:" $LOGFILE; then
    echo "✅ Handler registration detected"
    grep "Registered user response handler" $LOGFILE
else
    echo "❌ Handler registration not detected"
fi

# Check handler lookup
if grep -q "Handling user response for key:" $LOGFILE; then
    echo "✅ Handler lookup initiated"
else
    echo "❌ Handler lookup not initiated"
fi

# Check handler found/missing
if grep -q "Calling user response handler" $LOGFILE; then
    echo "✅ Handler found and called"
else
    echo "❌ Handler not found"
    if grep -q "No handler found for user response key" $LOGFILE; then
        echo "   Missing handler detected"
        grep "No handler found" $LOGFILE
    fi
fi

# Check final delivery
if grep -q "Personal Assistant:" $LOGFILE; then
    echo "✅ User received final response"
    echo ""
    echo "📝 Response Preview:"
    grep -A 2 "Personal Assistant:" $LOGFILE | head -3
else
    echo "❌ User did not receive response"
fi

# Check handler cleanup
if grep -q "Unregistered user response handler" $LOGFILE; then
    echo "✅ Handler cleanup detected"
else
    echo "ℹ️  Handler cleanup not logged"
fi

# Check for timing issues
echo ""
echo "⏱️  Timing Analysis:"
HANDLER_REG_TIME=$(grep "Registered user response handler" $LOGFILE | tail -1 | cut -d' ' -f1-2)
HANDLER_LOOKUP_TIME=$(grep "Handling user response for key" $LOGFILE | tail -1 | cut -d' ' -f1-2)

if [ ! -z "$HANDLER_REG_TIME" ] && [ ! -z "$HANDLER_LOOKUP_TIME" ]; then
    echo "   Registration: $HANDLER_REG_TIME"
    echo "   Lookup: $HANDLER_LOOKUP_TIME"
else
    echo "   Could not determine timing"
fi

echo ""
echo "🎯 Test Result:"
if grep -q "Personal Assistant:" $LOGFILE; then
    echo "🎉 SUCCESS: Handler lifecycle working correctly!"
    exit 0
else
    echo "❌ FAILED: Handler issues remain"
    echo "📋 Check log: $LOGFILE"
    exit 1
fi
