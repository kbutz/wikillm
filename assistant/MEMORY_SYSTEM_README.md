# Memory System Architecture and Verification

This document provides an overview of the memory system architecture in the WikiLLM assistant and explains how to verify that all memory aspects are working correctly.

## Memory System Architecture

The memory system consists of three main components:

### 1. Conversation Context (Fluid, Conversation-Level Memory)

The conversation context provides a fluid, conversation-level memory that allows the assistant to maintain context within a single conversation. This is implemented in the `ConversationManager` class, which:

- Stores messages in the database
- Retrieves recent messages for a conversation
- Builds a context array for the LLM that includes system context and conversation messages

The `build_conversation_context` method in `ConversationManager` is responsible for creating this context, which includes:
- System message with user memory and historical context
- Recent messages from the current conversation

### 2. Cross-Conversation RAG Pipeline

The cross-conversation RAG (Retrieval-Augmented Generation) pipeline allows the assistant to find relevant information from previous conversations. This is implemented in the `SearchManager` class, which:

- Extracts keywords from the current message
- Searches for related conversations using Full-Text Search (FTS) or a fallback LIKE search
- Returns conversation summaries that match the keywords

The `get_related_conversations` method in `SearchManager` is responsible for finding related conversations, which are then included in the historical context part of the conversation context.

### 3. Implicit and Explicit Memory Features

The memory features allow the assistant to store and retrieve individual pieces of data about a user. This is implemented in two classes:

#### MemoryManager (Original Implementation)

- Extracts implicit memories from user messages and assistant responses
- Extracts preferences from conversation patterns
- Extracts personal information from user messages
- Stores memories in the database
- Retrieves memory context for a user

#### EnhancedMemoryManager (New Implementation)

- Uses the original MemoryManager for backward compatibility
- Adds advanced entity extraction using LLM
- Categorizes memories as EXPLICIT (high confidence ≥ 0.8) or IMPLICIT (confidence between 0.6 and 0.8)
- Provides semantic search for memories
- Retrieves contextual memories based on the current message

## Database Schema

The memory system relies on several database tables:

- `conversations`: Stores conversation metadata
- `messages`: Stores individual messages in conversations
- `conversation_summaries`: Stores summaries of conversations for efficient search
- `conversation_summaries_fts`: Virtual FTS table for full-text search
- `user_memory`: Stores individual pieces of data about users
- `user_preferences`: Stores user preferences

## Verification

To verify that all memory aspects are working correctly, run the `verify_memory_system.py` script:

```bash
python verify_memory_system.py
```

This script performs the following tests:

1. **Conversation Context Test**: Creates a test conversation with multiple messages and verifies that the context contains all messages and maintains the correct order.

2. **Cross-Conversation RAG Test**: Creates two different conversations (one about Python, one about JavaScript) and verifies that the RAG pipeline can find the Python conversation when queried about Python exceptions.

3. **Implicit and Explicit Memory Test**: Tests both the original MemoryManager and the EnhancedMemoryManager to extract and store memories, and verifies that these memories can be retrieved in the memory context and as contextual memories.

The script also runs the enhanced database migration to ensure the database schema is up-to-date and the FTS table is populated.

## Common Issues and Fixes

### Database Schema Issues

If the database schema is missing tables or columns, run the enhanced migration script:

```bash
python enhanced_migration.py
```

This script:
- Adds missing columns to the conversations table (topic_tags)
- Adds missing columns to the conversation_summaries table (keywords, priority_score, updated_at)
- Creates an FTS virtual table and triggers for efficient text search
- Creates performance indexes for various tables
- Cleans up invalid memory entries
- Updates conversation summaries with missing data

### Memory Extraction Issues

If the memory extraction is not working correctly:

1. Check that the LLM client is configured correctly in `lmstudio_client.py`
2. Ensure that the extraction prompts in `memory_manager.py` are well-formed
3. Verify that the confidence thresholds are appropriate (0.6 for IMPLICIT, 0.8 for EXPLICIT)

### Search Issues

If the cross-conversation RAG pipeline is not working correctly:

1. Ensure that the FTS table is populated by running `enhanced_migration.py`
2. Check that the keyword extraction in `search_manager.py` is working correctly
3. Verify that conversation summaries are being created for all conversations

## Conclusion

The memory system in the WikiLLM assistant provides a comprehensive approach to maintaining context and retrieving relevant information. By combining conversation context, cross-conversation RAG, and implicit/explicit memory features, the assistant can provide personalized and contextually relevant responses.

The verification script ensures that all three memory aspects are working correctly, and the enhanced migration script fixes common database issues that could affect the memory system's functionality.