#!/usr/bin/env python3
"""
Debug Data Migration Script
Run this to fix existing debug data and ensure inline debug panels work properly
"""
import sys
import os
import logging

# Add the assistant directory to the path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from database import get_db
from debug_data_processor import DebugDataProcessor, migrate_existing_debug_data
from models import Message, DebugStep, LLMRequest

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def main():
    """Main migration function"""
    print("=== Debug Data Migration Script ===")
    print("This script will fix existing debug data to ensure inline debug panels work properly.")
    print()
    
    # Get database session
    db = next(get_db())
    
    try:
        # Initialize debug processor
        debug_processor = DebugDataProcessor(db)
        
        # Get statistics before migration
        total_messages = db.query(Message).count()
        debug_enabled_messages = db.query(Message).filter(Message.debug_enabled == True).count()
        messages_with_debug_data = db.query(Message).filter(
            Message.debug_enabled == True,
            Message.debug_data.isnot(None)
        ).count()
        
        debug_steps_count = db.query(DebugStep).count()
        llm_requests_count = db.query(LLMRequest).count()
        
        print(f"📊 Current Statistics:")
        print(f"   Total messages: {total_messages}")
        print(f"   Debug-enabled messages: {debug_enabled_messages}")
        print(f"   Messages with debug data: {messages_with_debug_data}")
        print(f"   Total debug steps: {debug_steps_count}")
        print(f"   Total LLM requests: {llm_requests_count}")
        print()
        
        # Find messages that need fixing
        messages_needing_fix = db.query(Message).filter(
            Message.debug_enabled == True,
            Message.debug_data.is_(None)
        ).count()
        
        print(f"🔧 Messages needing debug data fix: {messages_needing_fix}")
        print()
        
        if messages_needing_fix == 0:
            print("✅ No messages need fixing. Debug data is already properly configured.")
            return
        
        # Ask for confirmation
        response = input("Do you want to proceed with the migration? (y/N): ").strip().lower()
        if response != 'y':
            print("Migration cancelled.")
            return
        
        print("🚀 Starting migration...")
        
        # Run migration
        fixed_count = migrate_existing_debug_data(db)
        
        print(f"✅ Migration completed successfully!")
        print(f"   Fixed debug data for {fixed_count} messages")
        print()
        
        # Get statistics after migration
        messages_with_debug_data_after = db.query(Message).filter(
            Message.debug_enabled == True,
            Message.debug_data.isnot(None)
        ).count()
        
        print(f"📊 Updated Statistics:")
        print(f"   Messages with debug data: {messages_with_debug_data_after}")
        print(f"   Improvement: +{messages_with_debug_data_after - messages_with_debug_data}")
        print()
        
        print("🎉 Migration complete! Inline debug panels should now display data properly.")
        print()
        print("💡 To test the fix:")
        print("   1. Enable debug mode in the UI")
        print("   2. Send a message")
        print("   3. Look for debug information in the inline debug panel")
        print("   4. If you still see 'No Debug Data Found', try the migration endpoint:")
        print("      POST /debug/migrate-debug-data")
        
    except Exception as e:
        logger.error(f"Migration failed: {e}")
        print(f"❌ Migration failed: {e}")
        sys.exit(1)
    
    finally:
        db.close()


if __name__ == "__main__":
    main()
