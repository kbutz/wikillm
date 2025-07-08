#!/usr/bin/env python
"""
Verify Memory System Functionality

This script verifies that all three memory aspects are working correctly:
1. Conversation context (fluid, conversation-level memory)
2. Cross-conversation RAG pipeline (finding relevant information from previous chats)
3. "Implicit" and "explicit" memory features (storing individual pieces of data about a user)
"""
import logging
import sys
import os
import asyncio
from datetime import datetime
from sqlalchemy import text
from typing import List, Dict, Any

# Add current directory to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from database import get_db_session
from memory_manager import MemoryManager, EnhancedMemoryManager
from conversation_manager import ConversationManager
from search_manager import SearchManager
from models import User, Conversation, Message, UserMemory
from schemas import MessageRole, MemoryType, UserMemoryCreate

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

async def verify_conversation_context():
    """Verify that conversation context is working correctly"""
    logger.info("Verifying conversation context...")

    with get_db_session() as db:
        # Create a test user if needed
        user = db.query(User).filter(User.username == "test_user").first()
        if not user:
            user = User(username="test_user", email="test@example.com")
            db.add(user)
            db.commit()
            logger.info(f"Created test user with ID {user.id}")

        # Create a conversation manager
        conv_manager = ConversationManager(db)

        # Create a test conversation
        conversation = conv_manager.create_conversation(
            user_id=user.id,
            title="Test Conversation Context"
        )
        logger.info(f"Created test conversation with ID {conversation.id}")

        # Add some messages to the conversation
        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.USER,
            content="My name is Test User and I like pizza."
        )

        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.ASSISTANT,
            content="Nice to meet you, Test User! I'll remember that you like pizza."
        )

        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.USER,
            content="What's my favorite food?"
        )

        # Build conversation context
        context = await conv_manager.build_conversation_context(
            conversation_id=conversation.id,
            user_id=user.id
        )

        # Check if context contains all messages
        if len(context) < 4:  # System message + 3 conversation messages
            logger.error(f"Context missing messages: {context}")
            return False

        # Check if the last message is the user asking about favorite food
        if context[-1]["role"] != "user" or "favorite food" not in context[-1]["content"]:
            logger.error(f"Last message in context is incorrect: {context[-1]}")
            return False

        logger.info("Conversation context verified successfully")
        return True

async def verify_cross_conversation_rag():
    """Verify that cross-conversation RAG pipeline is working correctly"""
    logger.info("Verifying cross-conversation RAG pipeline...")

    with get_db_session() as db:
        # Get test user
        user = db.query(User).filter(User.username == "test_user").first()
        if not user:
            logger.error("Test user not found")
            return False

        # Create conversation manager and search manager
        conv_manager = ConversationManager(db)
        search_manager = SearchManager(db)

        # Create a first conversation with specific information
        conversation1 = conv_manager.create_conversation(
            user_id=user.id,
            title="Python Programming"
        )

        conv_manager.add_message(
            conversation_id=conversation1.id,
            role=MessageRole.USER,
            content="What's the best way to handle exceptions in Python?"
        )

        conv_manager.add_message(
            conversation_id=conversation1.id,
            role=MessageRole.ASSISTANT,
            content="In Python, you should use try/except blocks to handle exceptions. This allows your program to gracefully handle errors."
        )

        # Create a summary for the first conversation
        await conv_manager.create_conversation_summary(conversation1.id)

        # Create a second conversation with different information
        conversation2 = conv_manager.create_conversation(
            user_id=user.id,
            title="JavaScript Programming"
        )

        conv_manager.add_message(
            conversation_id=conversation2.id,
            role=MessageRole.USER,
            content="How do I handle promises in JavaScript?"
        )

        conv_manager.add_message(
            conversation_id=conversation2.id,
            role=MessageRole.ASSISTANT,
            content="In JavaScript, you can use .then() and .catch() methods or async/await syntax to handle promises."
        )

        # Create a summary for the second conversation
        await conv_manager.create_conversation_summary(conversation2.id)

        # Ensure FTS table is populated
        db.execute(text("INSERT INTO conversation_summaries_fts(conversation_summaries_fts) VALUES('rebuild')"))
        db.commit()

        # Now test the RAG pipeline with a query about Python
        related_conversations = await search_manager.get_related_conversations(
            user_id=user.id,
            message="I need help with Python exceptions"
        )

        # Check if the Python conversation is found
        python_found = False
        for conv in related_conversations:
            if conv.conversation_id == conversation1.id:
                python_found = True
                break

        if not python_found:
            logger.error(f"Python conversation not found in related conversations: {related_conversations}")
            return False

        logger.info("Cross-conversation RAG pipeline verified successfully")
        return True

async def verify_memory_features():
    """Verify that implicit and explicit memory features are working correctly"""
    logger.info("Verifying implicit and explicit memory features...")

    with get_db_session() as db:
        # Get test user
        user = db.query(User).filter(User.username == "test_user").first()
        if not user:
            logger.error("Test user not found")
            return False

        # Create memory managers
        memory_manager = MemoryManager(db)
        enhanced_memory = EnhancedMemoryManager(db)

        # Create a conversation manager
        conv_manager = ConversationManager(db)

        # Create a test conversation
        conversation = conv_manager.create_conversation(
            user_id=user.id,
            title="Test Memory Features"
        )

        # Test implicit memory extraction
        user_message = "I live in New York and I have a dog named Max."
        assistant_response = "That's great! New York is a wonderful city, and Max sounds like a lovely dog."

        # Extract implicit memories
        implicit_memories = memory_manager.extract_implicit_memory(
            user_id=user.id,
            message=user_message,
            response=assistant_response
        )

        # Store the memories
        stored_memories = memory_manager.store_memories(implicit_memories)

        # Check if memories were stored
        if not stored_memories:
            logger.error("No implicit memories were stored")
            return False

        # Test enhanced memory extraction
        user_message2 = "I work as a software engineer and I'm 32 years old."
        assistant_response2 = "Being a software engineer at 32 gives you a good balance of experience and energy."

        # Extract and store facts
        stored_facts = await enhanced_memory.extract_and_store_facts(
            user_id=user.id,
            user_message=user_message2,
            assistant_response=assistant_response2,
            conversation_id=conversation.id
        )

        # Manually add an explicit memory
        explicit_memory = UserMemoryCreate(
            user_id=user.id,
            memory_type=MemoryType.EXPLICIT,
            key="favorite_color",
            value="blue",
            confidence=0.95,
            source="direct_statement"
        )
        memory_manager.store_memory(explicit_memory)

        # Get memory context
        memory_context = memory_manager.get_memory_context(user.id)

        # Check if memory context contains our test memories
        if "New York" not in memory_context or "Max" not in memory_context:
            logger.error(f"Memory context missing implicit memories: {memory_context}")
            return False

        if "blue" not in memory_context:
            logger.error(f"Memory context missing explicit memory: {memory_context}")
            return False

        # Test contextual memories
        contextual_memories = await enhanced_memory.get_contextual_memories(
            user_id=user.id,
            current_message="Tell me about my dog"
        )

        if "Max" not in contextual_memories:
            logger.error(f"Contextual memories missing dog information: {contextual_memories}")
            return False

        logger.info("Implicit and explicit memory features verified successfully")
        return True

async def main():
    """Run all verification tests"""
    try:
        # Run database migration if needed
        try:
            from enhanced_migration import run_enhanced_migration, populate_fts_table
            run_enhanced_migration()
            populate_fts_table()
            logger.info("Database migration completed")
        except Exception as e:
            logger.warning(f"Could not run database migration: {e}")

        # Verify all memory aspects
        context_ok = await verify_conversation_context()
        rag_ok = await verify_cross_conversation_rag()
        memory_ok = await verify_memory_features()

        # Print summary
        print("\n=== Memory System Verification Results ===")
        print(f"Conversation Context: {'✓ PASS' if context_ok else '✗ FAIL'}")
        print(f"Cross-conversation RAG: {'✓ PASS' if rag_ok else '✗ FAIL'}")
        print(f"Implicit/Explicit Memory: {'✓ PASS' if memory_ok else '✗ FAIL'}")
        print("=========================================\n")

        if context_ok and rag_ok and memory_ok:
            print("✓ All memory aspects are working correctly!")
            return 0
        else:
            print("✗ Some memory aspects are not working correctly.")
            return 1

    except Exception as e:
        logger.error(f"Verification failed: {e}")
        print(f"✗ Verification failed: {e}")
        return 1

if __name__ == "__main__":
    asyncio.run(main())
