"""
Database Migration Script for Cross-Conversation Search
"""
import logging
from sqlalchemy import text
from database import get_db_session, engine
from models import Base

logger = logging.getLogger(__name__)


def migrate_database():
    """Apply database migrations for cross-conversation search"""
    logger.info("Starting database migration for cross-conversation search...")
    
    try:
        with get_db_session() as db:
            # Add new columns to existing tables
            logger.info("Adding new columns...")
            
            # Add topic_tags to conversations table
            try:
                db.execute(text("ALTER TABLE conversations ADD COLUMN topic_tags JSON"))
                logger.info("Added topic_tags column to conversations table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add topic_tags column: {e}")
            
            # Add new columns to conversation_summaries table
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN keywords TEXT"))
                logger.info("Added keywords column to conversation_summaries table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add keywords column: {e}")
            
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN priority_score REAL DEFAULT 0.0"))
                logger.info("Added priority_score column to conversation_summaries table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add priority_score column: {e}")
            
            try:
                db.execute(text("ALTER TABLE conversation_summaries ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP"))
                logger.info("Added updated_at column to conversation_summaries table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add updated_at column: {e}")
            
            # Add unique constraint to conversation_summaries
            try:
                db.execute(text("CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_summaries_conversation_id ON conversation_summaries(conversation_id)"))
                logger.info("Added unique index to conversation_summaries table")
            except Exception as e:
                logger.warning(f"Could not add unique index: {e}")
            
            # Create FTS virtual table for full-text search
            try:
                db.execute(text("""
                    CREATE VIRTUAL TABLE IF NOT EXISTS conversation_summaries_fts USING fts5(
                        summary, 
                        keywords,
                        content=conversation_summaries,
                        content_rowid=id
                    )
                """))
                logger.info("Created FTS virtual table")
                
                # Create triggers to keep FTS table in sync
                db.execute(text("""
                    CREATE TRIGGER IF NOT EXISTS conversation_summaries_ai AFTER INSERT ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                    END
                """))
                
                db.execute(text("""
                    CREATE TRIGGER IF NOT EXISTS conversation_summaries_ad AFTER DELETE ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                        VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                    END
                """))
                
                db.execute(text("""
                    CREATE TRIGGER IF NOT EXISTS conversation_summaries_au AFTER UPDATE ON conversation_summaries
                    BEGIN
                        INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                        VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                    END
                """))
                
                logger.info("Created FTS triggers")
            except Exception as e:
                logger.warning(f"Could not create FTS table or triggers: {e}")
            
            # Create indexes for better search performance
            try:
                db.execute(text("CREATE INDEX IF NOT EXISTS idx_conversations_user_active ON conversations(user_id, is_active)"))
                db.execute(text("CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at)"))
                db.execute(text("CREATE INDEX IF NOT EXISTS idx_conversation_summaries_priority ON conversation_summaries(priority_score)"))
                db.execute(text("CREATE INDEX IF NOT EXISTS idx_user_memory_key ON user_memory(user_id, key)"))
                logger.info("Created performance indexes")
            except Exception as e:
                logger.warning(f"Could not create indexes: {e}")
            
            db.commit()
            logger.info("Database migration completed successfully")
            
    except Exception as e:
        logger.error(f"Database migration failed: {e}")
        raise


def populate_existing_summaries():
    """Populate FTS table with existing summaries"""
    logger.info("Populating FTS table with existing summaries...")
    
    try:
        with get_db_session() as db:
            # Get all existing summaries
            result = db.execute(text("SELECT id, summary, keywords FROM conversation_summaries"))
            summaries = result.fetchall()
            
            # Insert into FTS table
            for summary in summaries:
                db.execute(text("""
                    INSERT OR REPLACE INTO conversation_summaries_fts(rowid, summary, keywords)
                    VALUES (:id, :summary, :keywords)
                """), {
                    "id": summary.id,
                    "summary": summary.summary,
                    "keywords": summary.keywords or ""
                })
            
            db.commit()
            logger.info(f"Populated FTS table with {len(summaries)} existing summaries")
            
    except Exception as e:
        logger.error(f"Failed to populate FTS table: {e}")


if __name__ == "__main__":
    # Setup logging
    logging.basicConfig(level=logging.INFO)
    
    # Run migration
    migrate_database()
    populate_existing_summaries()
    
    print("Database migration completed!")
