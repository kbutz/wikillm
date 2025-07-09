#!/usr/bin/env python3
"""
Test to verify that conversation title generation is properly removing thinking tags
"""
import asyncio
import os
import sys
import logging

# Add the current directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from database import init_database, get_db
from conversation_manager import ConversationManager
from models import User, Conversation, Message
from schemas import MessageRole

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def test_conversation_title_generation():
    """Test that conversation titles are properly generated without thinking tags"""
    
    print("Testing conversation title generation...")
    
    # Initialize database
    init_database()
    
    # Get database session
    db = next(get_db())
    
    try:
        # Create test user
        test_user = User(
            username="test_user_title",
            email="test@example.com",
            full_name="Test User"
        )
        db.add(test_user)
        db.commit()
        db.refresh(test_user)
        
        # Create conversation manager
        conv_manager = ConversationManager(db)
        
        # Create test conversation
        conversation = conv_manager.create_conversation(
            user_id=test_user.id,
            title="Test Conversation"
        )
        
        # Add test messages
        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.USER,
            content="Hello, can you help me with Python programming?"
        )
        
        conv_manager.add_message(
            conversation_id=conversation.id,
            role=MessageRole.ASSISTANT,
            content="Of course! I'd be happy to help you with Python programming. What specific aspect would you like to learn about?"
        )
        
        # Test title generation
        print("Generating conversation title...")
        title = await conv_manager.generate_conversation_title(conversation.id)
        
        print(f"Generated title: '{title}'")
        
        # Check if title contains thinking tags
        if title:
            if '<think>' in title or '</think>' in title:
                print("❌ FAILED: Title contains thinking tags!")
                print(f"   Title: {title}")
                return False
            else:
                print("✅ SUCCESS: Title is clean (no thinking tags)")
                print(f"   Title: {title}")
                return True
        else:
            print("❌ FAILED: No title generated")
            return False
    
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
    success = asyncio.run(test_conversation_title_generation())
    if success:
        print("\nTest passed! Conversation titles are being generated correctly.")
        sys.exit(0)
    else:
        print("\nTest failed! Check the logs for more details.")
        sys.exit(1)
