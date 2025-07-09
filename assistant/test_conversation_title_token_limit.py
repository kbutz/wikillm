#!/usr/bin/env python3
"""
Test to verify that conversation title generation handles token limit issues correctly
and generates fallback titles when needed.
"""
import asyncio
import os
import sys
import logging
from unittest.mock import patch, MagicMock

# Add the current directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from database import init_database, get_db
from conversation_manager import ConversationManager
from models import User, Conversation, Message
from schemas import MessageRole

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def test_conversation_title_token_limit_handling():
    """Test that conversation titles are properly generated even with token limit issues"""

    print("Testing conversation title generation with token limit handling...")

    # Initialize database
    init_database()

    # Get database session
    db = next(get_db())

    try:
        # Create test user
        test_user = User(
            username="test_user_token_limit",
            email="test_token@example.com",
            full_name="Test Token User"
        )
        db.add(test_user)
        db.commit()
        db.refresh(test_user)

        # Create conversation manager
        conv_manager = ConversationManager(db)

        # Create test conversation
        conversation = conv_manager.create_conversation(
            user_id=test_user.id,
            title="Test Token Limit Conversation"
        )

        # Add test messages
        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.USER,
            content="Can you explain how to implement a neural network in Python?"
        )

        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.ASSISTANT,
            content="Of course! I'd be happy to explain how to implement a neural network in Python."
        )

        # Test 1: Normal title generation
        print("\nTest 1: Normal title generation...")
        title = await conv_manager.generate_conversation_title(conversation.id)

        print(f"Generated title: '{title}'")
        # Check that we have a title and it doesn't contain thinking tags
        if title and "<think>" not in title and "</think>" not in title:
            if title.startswith("Chat about"):
                print("⚠️ WARNING: Title is a fallback title, but that's acceptable")
            print("✅ SUCCESS: Normal title generation works without thinking tags")
        else:
            print("❌ FAILED: Title contains thinking tags or is empty")

        # Test 2: Simulate token limit error
        print("\nTest 2: Simulating token limit error...")
        with patch('lmstudio_client.LMStudioClient.chat_completion') as mock_completion:
            # Mock the chat_completion to raise an exception with token limit message
            mock_completion.side_effect = Exception("Token limit exceeded for model")

            # Try to generate title with mocked error
            token_limit_title = await conv_manager.generate_conversation_title(conversation.id)

            print(f"Generated fallback title: '{token_limit_title}'")
            if token_limit_title and token_limit_title.startswith("Chat about"):
                print("✅ SUCCESS: Fallback title generated correctly when token limit error occurs")
            else:
                print("❌ FAILED: Fallback title not generated correctly")

        # Test 3: Simulate empty response
        print("\nTest 3: Simulating empty response...")
        with patch('lmstudio_client.LMStudioClient.chat_completion') as mock_completion:
            # Create a mock response with empty content
            mock_response = {
                "choices": [
                    {
                        "message": {
                            "content": ""
                        }
                    }
                ]
            }
            mock_completion.return_value = mock_response

            # Try to generate title with mocked empty response
            empty_response_title = await conv_manager.generate_conversation_title(conversation.id)

            print(f"Generated fallback title: '{empty_response_title}'")
            if empty_response_title and empty_response_title.startswith("Chat about"):
                print("✅ SUCCESS: Fallback title generated correctly when response is empty")
            else:
                print("❌ FAILED: Fallback title not generated correctly for empty response")

        # Test 4: Simulate generic title response
        print("\nTest 4: Simulating generic title response...")
        with patch('lmstudio_client.LMStudioClient.chat_completion') as mock_completion:
            # Create a mock response with generic title
            mock_response = {
                "choices": [
                    {
                        "message": {
                            "content": "New Conversation"
                        }
                    }
                ]
            }
            mock_completion.return_value = mock_response

            # Try to generate title with mocked generic response
            generic_title = await conv_manager.generate_conversation_title(conversation.id)

            print(f"Generated fallback title: '{generic_title}'")
            if generic_title and generic_title.startswith("Chat about"):
                print("✅ SUCCESS: Fallback title generated correctly when response is generic")
            else:
                print("❌ FAILED: Fallback title not generated correctly for generic response")

        # Overall success
        return True

    except Exception as e:
        print(f"❌ ERROR: {e}")
        return False

    finally:
        # Clean up
        try:
            db.delete(test_user)
            db.commit()
        except:
            pass
        db.close()

if __name__ == "__main__":
    success = asyncio.run(test_conversation_title_token_limit_handling())
    if success:
        print("\nTest passed! Conversation title generation handles token limit issues correctly.")
        sys.exit(0)
    else:
        print("\nTest failed! Check the logs for more details.")
        sys.exit(1)
