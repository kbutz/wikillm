#!/bin/bash

# Final test to verify user response delivery
echo "🎯 Final Test: User Response Delivery"
echo "===================================="

# Navigate to examples directory
cd "$(dirname "$0")"

# Clean memory state
echo "🧹 Cleaning memory..."
rm -rf wikillm_memory/memory/*

# Build
echo "🏗️  Building..."
go build -o interactive_example interactive_example.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# Test with communication query that should trigger specialist
echo "🚀 Testing with: 'help me write a professional email'"
LOGFILE="delivery_test.log"

(
    echo "help me write a professional email"
    sleep 25  # Give time for full coordination flow
    echo "exit"
) | timeout 120s ./interactive_example 2>&1 | tee $LOGFILE

echo ""
echo "📊 Analysis:"
echo "==========="

# Check coordination flow
if grep -q "CoordinationAgent: Extracted response key" $LOGFILE; then
    echo "✅ Response key extracted"
else
    echo "❌ Response key not extracted"
fi

if grep -q "CommunicationManagerAgent: Preserving coordination context" $LOGFILE; then
    echo "✅ Coordination context preserved"
else
    echo "❌ Coordination context not preserved"
fi

if grep -q "Orchestrator: Allowing coordination response" $LOGFILE; then
    echo "✅ Orchestrator allowed coordination response"
else
    echo "❌ Orchestrator blocked coordination response"
fi

if grep -q "CoordinatorAgent: Starting finalization" $LOGFILE; then
    echo "✅ Coordinator finalization started"
else
    echo "❌ Coordinator finalization did not start"
fi

if grep -q "CoordinatorAgent: Successfully sent final response" $LOGFILE; then
    echo "✅ Final response sent by coordinator"
else
    echo "❌ Final response not sent by coordinator"
fi

if grep -q "Personal Assistant:" $LOGFILE; then
    echo "✅ User received response"
    echo ""
    echo "📝 User Response:"
    echo "================="
    grep -A 10 "Personal Assistant:" $LOGFILE
else
    echo "❌ User did not receive response"
fi

# Check for loops
TOTAL_MESSAGES=$(grep -c "Routing message" $LOGFILE 2>/dev/null || echo 0)
echo ""
echo "📈 Total routed messages: $TOTAL_MESSAGES"

if [ $TOTAL_MESSAGES -gt 50 ]; then
    echo "⚠️  High message count - possible inefficiency"
elif [ $TOTAL_MESSAGES -lt 10 ]; then
    echo "⚠️  Low message count - coordination may not have occurred"
else
    echo "✅ Normal message count"
fi

echo ""
echo "🎯 Final Result:"
if grep -q "Personal Assistant:" $LOGFILE && [ $TOTAL_MESSAGES -lt 100 ]; then
    echo "🎉 SUCCESS: User response delivery is working!"
    exit 0
else
    echo "❌ FAILED: User response still not delivered"
    echo "📋 Log file: $LOGFILE"
    exit 1
fi
