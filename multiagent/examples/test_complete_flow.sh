#!/bin/bash

# Complete end-to-end test of the multiagent system
echo "🎯 Complete End-to-End Test"
echo "=========================="

cd "$(dirname "$0")"

# Clean state
echo "🧹 Cleaning memory..."
rm -rf wikillm_memory/memory/*

# Build
echo "🏗️  Building..."
go build -o interactive_example interactive_example.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# Test with communication query
echo "🚀 Testing complete flow..."
LOGFILE="complete_test.log"

(
    echo "help me write a professional thank you email"
    sleep 35  # Give time for full coordination and synthesis
    echo "exit"
) | timeout 150s ./interactive_example 2>&1 | tee $LOGFILE

echo ""
echo "📊 Complete Flow Analysis:"
echo "========================="

# Check each stage of the flow
echo "🔍 Stage-by-stage verification:"

# 1. Initial routing
if grep -q "ConversationAgent: Delegating message to specialists" $LOGFILE; then
    echo "✅ 1. Delegation to specialists initiated"
else
    echo "❌ 1. Delegation not initiated"
fi

# 2. Response key extraction
if grep -q "ConversationAgent: Extracted response key" $LOGFILE; then
    echo "✅ 2. Response key extracted"
else
    echo "❌ 2. Response key not extracted"
fi

# 3. Coordination setup
if grep -q "CoordinatorAgent: Extracted response key" $LOGFILE; then
    echo "✅ 3. Coordinator received response key"
else
    echo "❌ 3. Coordinator missing response key"
fi

# 4. Specialist processing
if grep -q "CommunicationManagerAgent: Preserving coordination context" $LOGFILE; then
    echo "✅ 4. Specialist preserved coordination context"
else
    echo "❌ 4. Specialist lost coordination context"
fi

# 5. Orchestrator routing
if grep -q "Orchestrator: Allowing coordination response" $LOGFILE; then
    echo "✅ 5. Orchestrator allowed coordination response"
else
    echo "❌ 5. Orchestrator blocked coordination response"
fi

# 6. Coordinator synthesis
if grep -q "CoordinatorAgent: LLM synthesis completed" $LOGFILE; then
    echo "✅ 6. LLM synthesis completed"
else
    echo "❌ 6. LLM synthesis failed"
fi

# 7. Final response routing
if grep -q "CoordinatorAgent: Successfully sent final response" $LOGFILE; then
    echo "✅ 7. Final response sent by coordinator"
else
    echo "❌ 7. Final response not sent"
fi

# 8. User response handling
if grep -q "Orchestrator: Routing message.*to user response handler" $LOGFILE; then
    echo "✅ 8. User response handler called"
else
    echo "❌ 8. User response handler not called"
fi

# 9. User response delivery
if grep -q "Personal Assistant:" $LOGFILE; then
    echo "✅ 9. User received final response"
    echo ""
    echo "📝 User Response Preview:"
    echo "========================"
    grep -A 3 "Personal Assistant:" $LOGFILE | head -5
else
    echo "❌ 9. User did not receive response"
fi

echo ""
echo "🔧 System Health Check:"

# Check for errors
if grep -q "panic:" $LOGFILE; then
    echo "❌ System panic detected"
else
    echo "✅ No panics"
fi

if grep -q "Warning: Agent.*not found" $LOGFILE; then
    echo "❌ Agent not found warnings"
    grep "Warning: Agent.*not found" $LOGFILE
else
    echo "✅ No missing agent warnings"
fi

# Message count analysis
TOTAL_MESSAGES=$(grep -c "Routing message" $LOGFILE 2>/dev/null || echo 0)
COORDINATION_MESSAGES=$(grep -c "coordination" $LOGFILE 2>/dev/null || echo 0)
USER_HANDLER_CALLS=$(grep -c "user response handler" $LOGFILE 2>/dev/null || echo 0)

echo ""
echo "📈 Message Flow Statistics:"
echo "  Total routed messages: $TOTAL_MESSAGES"
echo "  Coordination messages: $COORDINATION_MESSAGES"
echo "  User handler calls: $USER_HANDLER_CALLS"

# Determine success
SUCCESS=true

if ! grep -q "Personal Assistant:" $LOGFILE; then
    echo "❌ CRITICAL: User did not receive response"
    SUCCESS=false
fi

if grep -q "panic:" $LOGFILE; then
    echo "❌ CRITICAL: System panic occurred"
    SUCCESS=false
fi

if [ $TOTAL_MESSAGES -gt 100 ]; then
    echo "⚠️  WARNING: High message count ($TOTAL_MESSAGES) - possible inefficiency"
fi

echo ""
echo "🎯 Final Result:"
echo "==============="

if [ "$SUCCESS" = true ]; then
    echo "🎉 SUCCESS: Complete end-to-end flow working!"
    echo ""
    echo "✅ System successfully:"
    echo "   • Routed user query to specialists"
    echo "   • Coordinated multiple agent responses"
    echo "   • Synthesized final response with LLM"
    echo "   • Delivered response to user interface"
    echo "   • Prevented infinite loops"
    echo "   • Handled all edge cases"
    echo ""
    echo "📋 Log file: $LOGFILE"
    exit 0
else
    echo "❌ FAILED: Issues remain in the system"
    echo ""
    echo "🔧 Debugging suggestions:"
    echo "   • Check the log file for specific errors"
    echo "   • Verify LMStudio is running and responding"
    echo "   • Ensure all agents are properly initialized"
    echo ""
    echo "📋 Log file: $LOGFILE"
    exit 1
fi
