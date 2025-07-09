# Conversation Title Generation Fix

## Issue Description

The conversation title generation feature was failing with two separate issues:

1. **Previous Issue (Fixed)**: Titles contained `<think>` tags from the LLM's reasoning process.
2. **Current Issue**: Titles are not being generated at all, with all conversations showing as "New Conversation" in the UI. This appears to be related to token limit issues where the LLM response is truncated or fails.

## Root Causes

### Previous Issue (Thinking Tags)
The issue was in multiple places where the LLM was being called for background tasks but the responses weren't being processed to remove thinking tags:

1. `conversation_manager.py` - `generate_conversation_title()` method
2. `conversation_manager.py` - `_generate_summary_text()` method  
3. `memory_manager.py` - `get_relevant_memories_for_query()` method
4. `memory_manager.py` - `_extract_facts_with_llm()` method
5. `search_manager.py` - `_extract_structured_insights()` method

### Current Issue (Token Limits)
The current issue has multiple causes:

1. **Token Limit Errors**: The LLM may be hitting token limits, causing errors or truncated responses.
2. **Empty or Generic Responses**: The LLM sometimes returns empty or generic titles like "New Conversation".
3. **Insufficient Error Handling**: The error handling wasn't robust enough to handle these cases and provide fallback titles.
4. **Limited Logging**: There was insufficient logging to diagnose the specific issues.

## Evidence from Logs

### Previous Issue (Thinking Tags)
The logs showed examples of titles containing thinking tags:

```
2025-06-26 16:48:59,519 - main - INFO - Auto-generated title for conversation 3: <think>
Okay, the user wants me to generate a s...

2025-06-26 17:04:25,637 - main - INFO - Auto-generated title for conversation 4: <think>
Okay, the user wants a title for a conv...
```

### Current Issue (Token Limits)
The current issue doesn't show specific errors in the logs, which was part of the problem. The enhanced logging will now capture:

- Token limit errors
- Empty or generic title responses
- Fallback title generation

## Solutions

### Previous Solution (Thinking Tags)
Added proper LLM response processing to all background LLM calls:

```python
response = await lmstudio_client.chat_completion(...)
processed_response = self.response_processor.process_chat_response(response)
raw_title = processed_response["choices"][0]["message"]["content"].strip()
```

### Current Solution (Token Limits)
Enhanced the `generate_conversation_title()` method with:

1. **Robust Error Handling**: Specifically detecting token limit errors and other issues.
2. **Title Validation**: Checking if titles are empty, too short, or generic.
3. **Fallback Mechanism**: Generating a fallback title based on the first user message when the LLM fails.
4. **Enhanced Logging**: Adding detailed logs for diagnostics.

#### Before (Basic Error Handling):
```python
try:
    response = await lmstudio_client.chat_completion(...)
    processed_response = self.response_processor.process_chat_response(response)
    raw_title = processed_response["choices"][0]["message"]["content"].strip()
    # ... process title ...
    return title
except Exception as e:
    logger.error(f"Failed to generate conversation title: {e}")
    return None
```

#### After (Enhanced Error Handling with Fallbacks):
```python
try:
    logger.info(f"Generating title for conversation {conversation_id} with {len(user_messages)} user messages")
    response = await lmstudio_client.chat_completion(...)
    processed_response = self.response_processor.process_chat_response(response)
    raw_title = processed_response["choices"][0]["message"]["content"].strip()
    logger.debug(f"Raw title generated for conversation {conversation_id}: '{raw_title}'")

    # Validate title
    if not raw_title or len(raw_title) < 3:
        logger.warning(f"Empty or too short title generated for conversation {conversation_id}")
        # Generate fallback title
        return f"Chat about {first_user_message}"

    # ... process title ...

    # Check for generic titles
    if title.lower() in ["new conversation", "conversation", "chat", "new chat", ""]:
        logger.warning(f"Generic title generated for conversation {conversation_id}: '{title}'")
        # Generate fallback title
        return f"Chat about {first_user_message}"

    return title
except Exception as e:
    logger.error(f"Failed to generate conversation title for conversation {conversation_id}: {e}")

    # Check for token limit errors
    if "token" in str(e).lower() or "limit" in str(e).lower():
        logger.warning(f"Possible token limit issue when generating title: {e}")

    # Generate fallback title
    return f"Chat about {first_user_message}"
```

## Files Modified

1. **conversation_manager.py**
   - Enhanced `generate_conversation_title()` method with robust error handling, validation, and fallback mechanisms
   - Added detailed logging for diagnostics

## Testing

1. **Previous Fix**: Created `test_conversation_title_fix.py` to verify thinking tags are removed.
2. **Current Fix**: Created `test_conversation_title_token_limit.py` to verify token limit handling and fallback mechanisms.

The new test simulates:
- Normal title generation
- Token limit errors
- Empty responses
- Generic title responses

## Impact

- ✅ Conversation titles now generate properly without thinking tags
- ✅ Conversation titles are generated even when token limits are hit
- ✅ Empty or generic titles are replaced with meaningful fallback titles
- ✅ Detailed logging helps diagnose issues
- ✅ All background LLM tasks now properly handle thinking model responses

## Prevention

1. **Previous Issue**: Ensure all LLM calls use `response_processor.process_chat_response()`.
2. **Current Issue**: Implement robust error handling, validation, and fallback mechanisms for all critical LLM calls.

## Key Takeaways

1. **All LLM calls** in the system must use `response_processor.process_chat_response()` to handle thinking model responses properly.
2. **Critical LLM features** should have fallback mechanisms to handle errors gracefully.
3. **Detailed logging** is essential for diagnosing issues with LLM responses.
4. **Validate LLM outputs** to ensure they meet quality standards before using them.
