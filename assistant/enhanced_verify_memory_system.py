#!/usr/bin/env python
"""
Enhanced Memory System Verification with Detailed Logging

This script provides comprehensive testing of the memory system with:
1. Detailed step-by-step logging of all operations
2. Database health checks and FTS table verification
3. Multiple search strategies and fallback testing
4. Automated test environment setup and cleanup
"""
import logging
import sys
import os
import asyncio
import traceback
from datetime import datetime
from sqlalchemy import text
from typing import List, Dict, Any, Optional, Tuple

# Add current directory to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from database import get_db_session, get_db_direct, engine
from memory_manager import MemoryManager, EnhancedMemoryManager
from conversation_manager import ConversationManager
from search_manager import SearchManager
from models import User, Conversation, Message, UserMemory, ConversationSummary
from schemas import MessageRole, MemoryType, UserMemoryCreate

# Enhanced logging setup
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('memory_verification.log')
    ]
)
logger = logging.getLogger(__name__)

class MemoryVerificationTester:
    """Enhanced memory verification with detailed logging"""

    def __init__(self):
        self.test_user = None
        self.test_conversations = []
        self.test_results = {
            'database_health': False,
            'conversation_context': False,
            'cross_conversation_rag': False,
            'memory_features': False,
            'search_fallbacks': False
        }
        self.detailed_logs = []
        self.db = None

    def log_test_step(self, step: str, details: str = ""):
        """Log test step with details"""
        timestamp = datetime.now().isoformat()
        log_entry = f"[{timestamp}] {step}: {details}"
        self.detailed_logs.append(log_entry)
        logger.info(log_entry)

    async def setup(self):
        """Set up test environment"""
        self.log_test_step("SETUP", "Initializing test environment")

        try:
            # Get database session
            self.db = get_db_direct()

            # Create test user
            self.test_user = self.db.query(User).filter(User.username == "test_user").first()
            if not self.test_user:
                self.test_user = User(username="test_user", email="test@example.com")
                self.db.add(self.test_user)
                self.db.commit()
                self.log_test_step("SETUP", f"Created test user with ID {self.test_user.id}")
            else:
                self.log_test_step("SETUP", f"Using existing test user with ID {self.test_user.id}")

            # Run database health check
            await self.check_database_health()

            return True
        except Exception as e:
            self.log_test_step("SETUP ERROR", f"Failed to set up test environment: {e}")
            traceback.print_exc()
            return False

    async def check_database_health(self):
        """Check database health and FTS table status"""
        self.log_test_step("DATABASE", "Checking database health")

        try:
            # Import database health check function
            from enhanced_migration import check_database_health

            # Run health check
            issues = check_database_health()
            if issues:
                self.log_test_step("DATABASE WARNING", f"Database issues found: {issues}")

                # Try to fix issues by running migration
                self.log_test_step("DATABASE", "Attempting to fix issues with migration")
                from enhanced_migration import run_enhanced_migration, populate_fts_table
                run_enhanced_migration()
                populate_fts_table()

                # Check again
                issues = check_database_health()
                if issues:
                    self.log_test_step("DATABASE ERROR", f"Issues persist after migration: {issues}")
                    self.test_results['database_health'] = False
                else:
                    self.log_test_step("DATABASE", "Issues resolved after migration")
                    self.test_results['database_health'] = True
            else:
                self.log_test_step("DATABASE", "Database health check passed")
                self.test_results['database_health'] = True

            # Check FTS table specifically
            fts_count = self.db.execute(text("SELECT COUNT(*) FROM conversation_summaries_fts")).scalar()
            summary_count = self.db.execute(text("SELECT COUNT(*) FROM conversation_summaries")).scalar()

            self.log_test_step("DATABASE", f"FTS table has {fts_count} entries, summaries table has {summary_count} entries")

            if fts_count == 0 and summary_count > 0:
                self.log_test_step("DATABASE WARNING", "FTS table is empty but summaries exist")

                # Try to populate FTS table
                self.log_test_step("DATABASE", "Attempting to populate FTS table")
                from enhanced_migration import populate_fts_table
                populate_fts_table()

                # Check again
                fts_count = self.db.execute(text("SELECT COUNT(*) FROM conversation_summaries_fts")).scalar()
                self.log_test_step("DATABASE", f"FTS table now has {fts_count} entries")

            return self.test_results['database_health']

        except Exception as e:
            self.log_test_step("DATABASE ERROR", f"Database health check failed: {e}")
            traceback.print_exc()
            self.test_results['database_health'] = False
            return False

    async def verify_conversation_context(self):
        """Verify that conversation context is working correctly"""
        self.log_test_step("TEST", "Verifying conversation context")

        try:
            # Create a conversation manager
            conv_manager = ConversationManager(self.db)

            # Create a test conversation
            conversation = conv_manager.create_conversation(
                user_id=self.test_user.id,
                title="Test Conversation Context"
            )
            self.test_conversations.append(conversation.id)
            self.log_test_step("CONTEXT", f"Created test conversation with ID {conversation.id}")

            # Add some messages to the conversation
            self.log_test_step("CONTEXT", "Adding test messages")
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
            self.log_test_step("CONTEXT", "Building conversation context")
            context = await conv_manager.build_conversation_context(
                conversation_id=conversation.id,
                user_id=self.test_user.id
            )

            # Check if context contains all messages
            if len(context) < 4:  # System message + 3 conversation messages
                self.log_test_step("CONTEXT ERROR", f"Context missing messages: {context}")
                self.test_results['conversation_context'] = False
                return False

            # Check if the last message is the user asking about favorite food
            if context[-1]["role"] != "user" or "favorite food" not in context[-1]["content"]:
                self.log_test_step("CONTEXT ERROR", f"Last message in context is incorrect: {context[-1]}")
                self.test_results['conversation_context'] = False
                return False

            self.log_test_step("CONTEXT", "Conversation context verified successfully")
            self.test_results['conversation_context'] = True
            return True

        except Exception as e:
            self.log_test_step("CONTEXT ERROR", f"Conversation context verification failed: {e}")
            traceback.print_exc()
            self.test_results['conversation_context'] = False
            return False

    async def verify_cross_conversation_rag(self):
        """Verify that cross-conversation RAG pipeline is working correctly"""
        self.log_test_step("TEST", "Verifying cross-conversation RAG pipeline")

        try:
            # Create conversation manager and search manager
            conv_manager = ConversationManager(self.db)
            search_manager = SearchManager(self.db)

            # Create a first conversation with specific information
            self.log_test_step("RAG", "Creating first test conversation about Python")
            conversation1 = conv_manager.create_conversation(
                user_id=self.test_user.id,
                title="Python Programming"
            )
            self.test_conversations.append(conversation1.id)

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
            self.log_test_step("RAG", "Creating summary for first conversation")
            await conv_manager.create_conversation_summary(conversation1.id)

            # Create a second conversation with different information
            self.log_test_step("RAG", "Creating second test conversation about JavaScript")
            conversation2 = conv_manager.create_conversation(
                user_id=self.test_user.id,
                title="JavaScript Programming"
            )
            self.test_conversations.append(conversation2.id)

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
            self.log_test_step("RAG", "Creating summary for second conversation")
            await conv_manager.create_conversation_summary(conversation2.id)

            # Ensure FTS table is populated
            self.log_test_step("RAG", "Ensuring FTS table is populated")
            try:
                self.db.execute(text("INSERT INTO conversation_summaries_fts(conversation_summaries_fts) VALUES('rebuild')"))
                self.db.commit()
            except Exception as e:
                self.log_test_step("RAG WARNING", f"FTS rebuild command failed: {e}")
                # Try alternative approach
                from enhanced_migration import populate_fts_table
                populate_fts_table()

            # Now test the RAG pipeline with a query about Python
            self.log_test_step("RAG", "Testing search with Python query")
            related_conversations = await search_manager.get_related_conversations(
                user_id=self.test_user.id,
                message="I need help with Python exceptions"
            )

            # Check if the Python conversation is found
            python_found = False
            for conv in related_conversations:
                if conv.conversation_id == conversation1.id:
                    python_found = True
                    break

            if not python_found:
                self.log_test_step("RAG ERROR", f"Python conversation not found in related conversations: {related_conversations}")
                self.test_results['cross_conversation_rag'] = False
                return False

            self.log_test_step("RAG", "Cross-conversation RAG pipeline verified successfully")
            self.test_results['cross_conversation_rag'] = True
            return True

        except Exception as e:
            self.log_test_step("RAG ERROR", f"Cross-conversation RAG verification failed: {e}")
            traceback.print_exc()
            self.test_results['cross_conversation_rag'] = False
            return False

    async def verify_search_fallbacks(self):
        """Verify that search fallbacks work correctly"""
        self.log_test_step("TEST", "Verifying search fallbacks")

        try:
            # Create search manager
            search_manager = SearchManager(self.db)

            # Test FTS search
            self.log_test_step("FALLBACK", "Testing FTS search")
            fts_results = await search_manager._search_with_fts(
                user_id=self.test_user.id,
                query="python exceptions",
                limit=5
            )
            self.log_test_step("FALLBACK", f"FTS search returned {len(fts_results)} results")

            # Test SQL search
            self.log_test_step("FALLBACK", "Testing SQL search")
            sql_results = await search_manager._search_with_sql(
                user_id=self.test_user.id,
                query="python exceptions",
                limit=5
            )
            self.log_test_step("FALLBACK", f"SQL search returned {len(sql_results)} results")

            # Test title search
            self.log_test_step("FALLBACK", "Testing title search")
            title_results = await search_manager._search_by_title(
                user_id=self.test_user.id,
                query="python",
                limit=5
            )
            self.log_test_step("FALLBACK", f"Title search returned {len(title_results)} results")

            # Test content search
            self.log_test_step("FALLBACK", "Testing content search")
            content_results = await search_manager._search_by_content(
                user_id=self.test_user.id,
                query="exceptions",
                limit=5
            )
            self.log_test_step("FALLBACK", f"Content search returned {len(content_results)} results")

            # Verify at least one search method works
            if len(fts_results) > 0 or len(sql_results) > 0 or len(title_results) > 0 or len(content_results) > 0:
                self.log_test_step("FALLBACK", "At least one search method is working")
                self.test_results['search_fallbacks'] = True
                return True
            else:
                self.log_test_step("FALLBACK ERROR", "All search methods failed")
                self.test_results['search_fallbacks'] = False
                return False

        except Exception as e:
            self.log_test_step("FALLBACK ERROR", f"Search fallback verification failed: {e}")
            traceback.print_exc()
            self.test_results['search_fallbacks'] = False
            return False

    async def verify_memory_features(self):
        """Verify that implicit and explicit memory features are working correctly"""
        self.log_test_step("TEST", "Verifying implicit and explicit memory features")

        try:
            # Create memory managers
            memory_manager = MemoryManager(self.db)
            enhanced_memory = EnhancedMemoryManager(self.db)

            # Create a conversation manager
            conv_manager = ConversationManager(self.db)

            # Create a test conversation
            self.log_test_step("MEMORY", "Creating test conversation for memory features")
            conversation = conv_manager.create_conversation(
                user_id=self.test_user.id,
                title="Test Memory Features"
            )
            self.test_conversations.append(conversation.id)

            # Test implicit memory extraction
            self.log_test_step("MEMORY", "Testing implicit memory extraction")
            user_message = "I live in New York and I have a dog named Max."
            assistant_response = "That's great! New York is a wonderful city, and Max sounds like a lovely dog."

            # Extract implicit memories
            implicit_memories = memory_manager.extract_implicit_memory(
                user_id=self.test_user.id,
                message=user_message,
                response=assistant_response
            )

            self.log_test_step("MEMORY", f"Extracted {len(implicit_memories)} implicit memories")

            # Store the memories
            stored_memories = memory_manager.store_memories(implicit_memories)
            self.log_test_step("MEMORY", f"Stored {len(stored_memories)} implicit memories")

            # Check if memories were stored
            if not stored_memories:
                self.log_test_step("MEMORY ERROR", "No implicit memories were stored")
                self.test_results['memory_features'] = False
                return False

            # Test enhanced memory extraction
            self.log_test_step("MEMORY", "Testing enhanced memory extraction")
            user_message2 = "I work as a software engineer and I'm 32 years old."
            assistant_response2 = "Being a software engineer at 32 gives you a good balance of experience and energy."

            # Extract and store facts
            stored_facts = await enhanced_memory.extract_and_store_facts(
                user_id=self.test_user.id,
                user_message=user_message2,
                assistant_response=assistant_response2,
                conversation_id=conversation.id
            )
            self.log_test_step("MEMORY", f"Extracted and stored {len(stored_facts) if stored_facts else 0} enhanced facts")

            # Manually add an explicit memory
            self.log_test_step("MEMORY", "Adding explicit memory")
            explicit_memory = UserMemoryCreate(
                user_id=self.test_user.id,
                memory_type=MemoryType.EXPLICIT,
                key="favorite_color",
                value="blue",
                confidence=0.95,
                source="direct_statement"
            )
            memory_manager.store_memory(explicit_memory)

            # Get memory context
            self.log_test_step("MEMORY", "Retrieving memory context")
            memory_context = memory_manager.get_memory_context(self.test_user.id)
            self.log_test_step("MEMORY", f"Memory context: {memory_context}")

            # Check if memory context contains our test memories
            if "New York" not in memory_context or "Max" not in memory_context:
                self.log_test_step("MEMORY ERROR", f"Memory context missing implicit memories: {memory_context}")
                self.test_results['memory_features'] = False
                return False

            if "blue" not in memory_context:
                self.log_test_step("MEMORY ERROR", f"Memory context missing explicit memory: {memory_context}")
                self.test_results['memory_features'] = False
                return False

            # Test contextual memories
            self.log_test_step("MEMORY", "Testing contextual memory retrieval")
            contextual_memories = await enhanced_memory.get_contextual_memories(
                user_id=self.test_user.id,
                current_message="Tell me about my dog"
            )
            self.log_test_step("MEMORY", f"Contextual memories: {contextual_memories}")

            if "Max" not in contextual_memories:
                self.log_test_step("MEMORY ERROR", f"Contextual memories missing dog information: {contextual_memories}")
                self.test_results['memory_features'] = False
                return False

            self.log_test_step("MEMORY", "Implicit and explicit memory features verified successfully")
            self.test_results['memory_features'] = True
            return True

        except Exception as e:
            self.log_test_step("MEMORY ERROR", f"Memory features verification failed: {e}")
            traceback.print_exc()
            self.test_results['memory_features'] = False
            return False

    async def cleanup(self):
        """Clean up test data"""
        self.log_test_step("CLEANUP", "Cleaning up test data")

        try:
            # Delete test conversations
            for conv_id in self.test_conversations:
                try:
                    self.log_test_step("CLEANUP", f"Deleting test conversation {conv_id}")
                    self.db.execute(text("DELETE FROM messages WHERE conversation_id = :conv_id"), {"conv_id": conv_id})
                    self.db.execute(text("DELETE FROM conversation_summaries WHERE conversation_id = :conv_id"), {"conv_id": conv_id})
                    self.db.execute(text("DELETE FROM conversations WHERE id = :conv_id"), {"conv_id": conv_id})
                except Exception as e:
                    self.log_test_step("CLEANUP WARNING", f"Failed to delete conversation {conv_id}: {e}")

            # Delete test user memories
            try:
                self.log_test_step("CLEANUP", f"Deleting test user memories for user {self.test_user.id}")
                self.db.execute(text("DELETE FROM user_memory WHERE user_id = :user_id"), {"user_id": self.test_user.id})
            except Exception as e:
                self.log_test_step("CLEANUP WARNING", f"Failed to delete user memories: {e}")

            self.db.commit()
            self.log_test_step("CLEANUP", "Test data cleanup completed")
            return True

        except Exception as e:
            self.log_test_step("CLEANUP ERROR", f"Test data cleanup failed: {e}")
            traceback.print_exc()
            return False

    async def run_all_tests(self):
        """Run all verification tests"""
        self.log_test_step("START", "Starting memory system verification")

        try:
            # Setup test environment
            setup_ok = await self.setup()
            if not setup_ok:
                self.log_test_step("ERROR", "Test setup failed, aborting tests")
                return False

            # Run all tests
            await self.verify_conversation_context()
            await self.verify_cross_conversation_rag()
            await self.verify_search_fallbacks()
            await self.verify_memory_features()

            # Clean up
            await self.cleanup()

            # Print summary
            self.log_test_step("SUMMARY", f"Test results: {self.test_results}")

            # Return overall success
            return all(self.test_results.values())

        except Exception as e:
            self.log_test_step("ERROR", f"Test execution failed: {e}")
            traceback.print_exc()
            return False

async def main():
    """Main verification function"""
    print("\n=== Enhanced Memory System Verification ===")
    print("Running comprehensive tests with detailed logging...\n")

    try:
        tester = MemoryVerificationTester()
        success = await tester.run_all_tests()

        # Print summary
        print("\n=== Memory System Verification Results ===")
        for test, result in tester.test_results.items():
            print(f"{test.replace('_', ' ').title()}: {'✓ PASS' if result else '✗ FAIL'}")
        print("=========================================\n")

        if success:
            print("✓ All memory aspects are working correctly!")
            return 0
        else:
            print("✗ Some memory aspects are not working correctly.")
            print("Check memory_verification.log for detailed information.")
            return 1

    except Exception as e:
        logger.error(f"Verification failed: {e}")
        traceback.print_exc()
        print(f"✗ Verification failed: {e}")
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
