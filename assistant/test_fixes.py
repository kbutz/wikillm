#!/usr/bin/env python3
"""
Test script to verify the fixes are working
"""
import sys
import os
sys.path.append('/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')

def test_imports():
    """Test that all modules can be imported"""
    try:
        from models import User, Conversation, Message, ConversationSummary
        print("✓ Models imported successfully")
        
        from search_manager import SearchManager
        print("✓ SearchManager imported successfully")
        
        from conversation_manager import ConversationManager
        print("✓ ConversationManager imported successfully")
        
        from database import get_db_session
        print("✓ Database module imported successfully")
        
        return True
    except Exception as e:
        print(f"❌ Import error: {e}")
        import traceback
        traceback.print_exc()
        return False

def test_database_connection():
    """Test database connection"""
    try:
        from database import get_db_session
        
        with get_db_session() as db:
            print("✓ Database connection successful")
            
            # Test creating managers
            from search_manager import SearchManager
            search_manager = SearchManager(db)
            print("✓ SearchManager created successfully")
            
            from conversation_manager import ConversationManager
            conv_manager = ConversationManager(db)
            print("✓ ConversationManager created successfully")
            
        return True
    except Exception as e:
        print(f"❌ Database connection error: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    print("Testing assistant application fixes...")
    print("=" * 50)
    
    # Test 1: Imports
    print("\n1. Testing imports...")
    if not test_imports():
        return False
    
    # Test 2: Database connection
    print("\n2. Testing database connection...")
    if not test_database_connection():
        return False
    
    print("\n" + "=" * 50)
    print("✅ All tests passed! The application should work now.")
    return True

if __name__ == "__main__":
    success = main()
    if not success:
        sys.exit(1)
