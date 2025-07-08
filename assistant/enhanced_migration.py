#!/usr/bin/env python
"""
Enhanced Database Migration with Memory System Fixes
"""
import logging
import sys
import os
from datetime import datetime
from sqlalchemy import text, create_engine, inspect
from sqlalchemy.orm import sessionmaker

# Add current directory to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from config import settings
from models import Base, User, Conversation, ConversationSummary, UserMemory, UserPreference
from database import get_db_session, engine

logger = logging.getLogger(__name__)


def check_database_health():
    """Check current database health and identify issues"""
    logger.info("Checking database health...")
    
    issues = []
    
    try:
        with get_db_session() as db:
            # Check if all required tables exist
            inspector = inspect(engine)
            existing_tables = inspector.get_table_names()
            
            required_tables = ['users', 'conversations', 'messages', 'user_memory', 'user_preferences', 'conversation_summaries']
            missing_tables = [table for table in required_tables if table not in existing_tables]
            
            if missing_tables:
                issues.append(f"Missing tables: {missing_tables}")
            
            # Check conversations table schema
            if 'conversations' in existing_tables:
                conversation_columns = [col['name'] for col in inspector.get_columns('conversations')]
                if 'topic_tags' not in conversation_columns:
                    issues.append("conversations table missing topic_tags column")
            
            # Check conversation_summaries table schema
            if 'conversation_summaries' in existing_tables:
                summary_columns = [col['name'] for col in inspector.get_columns('conversation_summaries')]
                required_summary_columns = ['keywords', 'priority_score', 'updated_at']
                missing_summary_columns = [col for col in required_summary_columns if col not in summary_columns]
                if missing_summary_columns:
                    issues.append(f"conversation_summaries table missing columns: {missing_summary_columns}")
            
            # Check if FTS table exists
            fts_exists = db.execute(text("SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_summaries_fts'")).fetchone()
            if not fts_exists:
                issues.append("FTS table missing")
            
            # Check data integrity
            user_count = db.execute(text("SELECT COUNT(*) FROM users")).scalar()
            conversation_count = db.execute(text("SELECT COUNT(*) FROM conversations")).scalar()
            memory_count = db.execute(text("SELECT COUNT(*) FROM user_memory")).scalar()
            
            logger.info(f"Database stats: {user_count} users, {conversation_count} conversations, {memory_count} memories")
            
    except Exception as e:
        issues.append(f"Database access error: {e}")
    
    return issues


def run_enhanced_migration():
    """Run enhanced database migration with all fixes"""
    logger.info("Starting enhanced database migration...")
    
    try:
        # First, ensure all tables exist
        Base.metadata.create_all(bind=engine)
        logger.info("Ensured all tables exist")
        
        with get_db_session() as db:
            # 1. Add missing columns to conversations table
            logger.info("Adding missing columns to conversations table...")
            try:
                db.execute(text("ALTER TABLE conversations ADD COLUMN topic_tags JSON"))
                logger.info("Added topic_tags column")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add topic_tags column: {e}")
            
            # 2. Add missing columns to conversation_summaries table
            logger.info("Adding missing columns to conversation_summaries table...")
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN keywords TEXT"))
                logger.info("Added keywords column")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add keywords column: {e}")
            
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN priority_score REAL DEFAULT 0.0"))
                logger.info("Added priority_score column")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add priority_score column: {e}")
            
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP"))
                logger.info("Added updated_at column")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add updated_at column: {e}")
            
            # 3. Create FTS virtual table and triggers
            logger.info("Creating FTS virtual table...")
            try:
                # Drop existing FTS table if it exists to recreate it properly
                db.execute(text("DROP TABLE IF EXISTS conversation_summaries_fts"))
                
                # Create FTS table
                db.execute(text("""
                    CREATE VIRTUAL TABLE conversation_summaries_fts USING fts5(
                        summary, 
                        keywords,
                        content=conversation_summaries,
                        content_rowid=id
                    )
                """))
                logger.info("Created FTS virtual table")
                
                # Drop existing triggers
                db.execute(text("DROP TRIGGER IF EXISTS conversation_summaries_ai"))
                db.execute(text("DROP TRIGGER IF EXISTS conversation_summaries_ad"))
                db.execute(text("DROP TRIGGER IF EXISTS conversation_summaries_au"))
                
                # Create triggers to keep FTS table in sync
                db.execute(text("""
                    CREATE TRIGGER conversation_summaries_ai AFTER INSERT ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                    END
                """))
                
                db.execute(text("""
                    CREATE TRIGGER conversation_summaries_ad AFTER DELETE ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                        VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                    END
                """))
                
                db.execute(text("""
                    CREATE TRIGGER conversation_summaries_au AFTER UPDATE ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                        VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                    END
                """))
                
                logger.info("Created FTS triggers")
            except Exception as e:
                logger.error(f"Could not create FTS table or triggers: {e}")
            
            # 4. Create performance indexes
            logger.info("Creating performance indexes...")
            indexes = [
                "CREATE INDEX IF NOT EXISTS idx_conversations_user_active ON conversations(user_id, is_active)",
                "CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at)",
                "CREATE INDEX IF NOT EXISTS idx_conversation_summaries_priority ON conversation_summaries(priority_score)",
                "CREATE INDEX IF NOT EXISTS idx_user_memory_key ON user_memory(user_id, key)",
                "CREATE INDEX IF NOT EXISTS idx_user_memory_type ON user_memory(user_id, memory_type)",
                "CREATE INDEX IF NOT EXISTS idx_user_memory_confidence ON user_memory(user_id, confidence)",
                "CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, timestamp)",
                "CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_summaries_conversation_id ON conversation_summaries(conversation_id)"
            ]
            
            for index_sql in indexes:
                try:
                    db.execute(text(index_sql))
                except Exception as e:
                    logger.warning(f"Could not create index: {e}")
            
            logger.info("Created performance indexes")
            
            # 5. Clean up invalid memory entries
            logger.info("Cleaning up invalid memory entries...")
            try:
                # Find and fix memory entries with invalid values
                invalid_memories = db.execute(text("""
                    SELECT id, key, value FROM user_memory 
                    WHERE value IS NULL OR value = '' OR TYPEOF(value) != 'text'
                """)).fetchall()
                
                for memory in invalid_memories:
                    # Convert non-string values to strings
                    if memory.value is None:
                        new_value = "unknown"
                    else:
                        new_value = str(memory.value)
                    
                    db.execute(text("""
                        UPDATE user_memory SET value = :new_value WHERE id = :id
                    """), {"new_value": new_value, "id": memory.id})
                
                logger.info(f"Fixed {len(invalid_memories)} invalid memory entries")
            except Exception as e:
                logger.warning(f"Could not clean up memory entries: {e}")
            
            # 6. Update conversation summaries with missing data
            logger.info("Updating conversation summaries...")
            try:
                # Set default values for missing fields
                db.execute(text("""
                    UPDATE conversation_summaries 
                    SET keywords = COALESCE(keywords, ''), 
                        priority_score = COALESCE(priority_score, 0.0),
                        updated_at = COALESCE(updated_at, CURRENT_TIMESTAMP)
                    WHERE keywords IS NULL OR priority_score IS NULL OR updated_at IS NULL
                """))
                logger.info("Updated conversation summaries with default values")
            except Exception as e:
                logger.warning(f"Could not update conversation summaries: {e}")
            
            db.commit()
            logger.info("Enhanced database migration completed successfully")
            
    except Exception as e:
        logger.error(f"Enhanced database migration failed: {e}")
        raise


def populate_fts_table():
    """Populate FTS table with existing summaries"""
    logger.info("Populating FTS table with existing summaries...")
    
    try:
        with get_db_session() as db:
            # Get all existing summaries
            result = db.execute(text("SELECT id, summary, keywords FROM conversation_summaries"))
            summaries = result.fetchall()
            
            # Clear existing FTS entries
            db.execute(text("DELETE FROM conversation_summaries_fts"))
            
            # Insert into FTS table
            for summary in summaries:
                try:
                    db.execute(text("""
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (:id, :summary, :keywords)
                    """), {
                        "id": summary.id,
                        "summary": summary.summary or "",
                        "keywords": summary.keywords or ""
                    })
                except Exception as e:
                    logger.warning(f"Could not insert summary {summary.id} into FTS: {e}")
            
            db.commit()
            logger.info(f"Populated FTS table with {len(summaries)} existing summaries")
            
    except Exception as e:
        logger.error(f"Failed to populate FTS table: {e}")


def verify_migration():
    """Verify that the migration was successful"""
    logger.info("Verifying migration...")
    
    try:
        with get_db_session() as db:
            # Check FTS table
            fts_count = db.execute(text("SELECT COUNT(*) FROM conversation_summaries_fts")).scalar()
            summary_count = db.execute(text("SELECT COUNT(*) FROM conversation_summaries")).scalar()
            
            logger.info(f"FTS table has {fts_count} entries, summaries table has {summary_count} entries")
            
            # Test FTS search
            test_results = db.execute(text("""
                SELECT COUNT(*) FROM conversation_summaries_fts 
                WHERE conversation_summaries_fts MATCH 'test' LIMIT 1
            """)).scalar()
            
            logger.info("FTS search test completed successfully")
            
            # Check schema
            inspector = inspect(engine)
            conversation_columns = [col['name'] for col in inspector.get_columns('conversations')]
            summary_columns = [col['name'] for col in inspector.get_columns('conversation_summaries')]
            
            logger.info(f"Conversations table columns: {conversation_columns}")
            logger.info(f"Conversation summaries table columns: {summary_columns}")
            
            # Check for required columns
            required_checks = [
                ('topic_tags' in conversation_columns, "conversations.topic_tags"),
                ('keywords' in summary_columns, "conversation_summaries.keywords"),
                ('priority_score' in summary_columns, "conversation_summaries.priority_score"),
                ('updated_at' in summary_columns, "conversation_summaries.updated_at")
            ]
            
            for check, name in required_checks:
                if check:
                    logger.info(f"✓ {name} column exists")
                else:
                    logger.error(f"✗ {name} column missing")
            
    except Exception as e:
        logger.error(f"Verification failed: {e}")


if __name__ == "__main__":
    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    try:
        # Check current database health
        issues = check_database_health()
        if issues:
            logger.warning(f"Database issues found: {issues}")
        else:
            logger.info("Database health check passed")
        
        # Run enhanced migration
        run_enhanced_migration()
        
        # Populate FTS table
        populate_fts_table()
        
        # Verify migration
        verify_migration()
        
        print("✓ Enhanced database migration completed successfully!")
        
    except Exception as e:
        logger.error(f"Migration failed: {e}")
        print(f"✗ Migration failed: {e}")
        sys.exit(1)
