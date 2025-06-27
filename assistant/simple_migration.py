"""
Simple Database Migration Script for Cross-Conversation Search
"""
import sqlite3
import logging
import os

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def run_migration():
    """Apply database migrations for cross-conversation search"""
    db_path = '/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant/assistant.db'
    
    if not os.path.exists(db_path):
        logger.error(f"Database not found at {db_path}")
        return False
    
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        logger.info("Starting database migration...")
        
        # Add new columns to existing tables
        try:
            cursor.execute("ALTER TABLE conversations ADD COLUMN topic_tags TEXT")
            logger.info("Added topic_tags column to conversations table")
        except sqlite3.OperationalError as e:
            if "duplicate column name" not in str(e).lower():
                logger.warning(f"Could not add topic_tags column: {e}")
        
        try:
            cursor.execute("ALTER TABLE conversation_summaries ADD COLUMN keywords TEXT")
            logger.info("Added keywords column to conversation_summaries table")
        except sqlite3.OperationalError as e:
            if "duplicate column name" not in str(e).lower():
                logger.warning(f"Could not add keywords column: {e}")
        
        try:
            cursor.execute("ALTER TABLE conversation_summaries ADD COLUMN priority_score REAL DEFAULT 0.0")
            logger.info("Added priority_score column to conversation_summaries table")
        except sqlite3.OperationalError as e:
            if "duplicate column name" not in str(e).lower():
                logger.warning(f"Could not add priority_score column: {e}")
        
        try:
            cursor.execute("ALTER TABLE conversation_summaries ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP")
            logger.info("Added updated_at column to conversation_summaries table")
        except sqlite3.OperationalError as e:
            if "duplicate column name" not in str(e).lower():
                logger.warning(f"Could not add updated_at column: {e}")
        
        # Update Message table to use llm_model instead of model_used
        try:
            cursor.execute("ALTER TABLE messages ADD COLUMN llm_model TEXT")
            logger.info("Added llm_model column to messages table")
        except sqlite3.OperationalError as e:
            if "duplicate column name" not in str(e).lower():
                logger.warning(f"Could not add llm_model column: {e}")
        
        # Create indexes for better performance
        try:
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_conversations_user_active ON conversations(user_id, is_active)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_conversation_summaries_priority ON conversation_summaries(priority_score)")
            cursor.execute("CREATE INDEX IF NOT EXISTS idx_user_memory_key ON user_memory(user_id, key)")
            logger.info("Created performance indexes")
        except sqlite3.OperationalError as e:
            logger.warning(f"Could not create indexes: {e}")
        
        # Create FTS virtual table for full-text search
        try:
            cursor.execute("""
                CREATE VIRTUAL TABLE IF NOT EXISTS conversation_summaries_fts USING fts5(
                    summary, 
                    keywords,
                    content=conversation_summaries,
                    content_rowid=id
                )
            """)
            logger.info("Created FTS virtual table")
        except sqlite3.OperationalError as e:
            logger.warning(f"Could not create FTS table: {e}")
        
        # Create FTS triggers
        try:
            cursor.execute("""
                CREATE TRIGGER IF NOT EXISTS conversation_summaries_ai AFTER INSERT ON conversation_summaries
                BEGIN
                    INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                    VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                END
            """)
            
            cursor.execute("""
                CREATE TRIGGER IF NOT EXISTS conversation_summaries_ad AFTER DELETE ON conversation_summaries
                BEGIN
                    INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                    VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                END
            """)
            
            cursor.execute("""
                CREATE TRIGGER IF NOT EXISTS conversation_summaries_au AFTER UPDATE ON conversation_summaries
                BEGIN
                    INSERT INTO conversation_summaries_fts(conversation_summaries_fts, rowid, summary, keywords)
                    VALUES ('delete', old.id, old.summary, COALESCE(old.keywords, ''));
                    INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                    VALUES (new.id, new.summary, COALESCE(new.keywords, ''));
                END
            """)
            
            logger.info("Created FTS triggers")
        except sqlite3.OperationalError as e:
            logger.warning(f"Could not create FTS triggers: {e}")
        
        # Commit all changes
        conn.commit()
        logger.info("Database migration completed successfully")
        
        # Populate existing summaries into FTS table
        try:
            cursor.execute("SELECT id, summary, keywords FROM conversation_summaries")
            summaries = cursor.fetchall()
            
            for summary_id, summary, keywords in summaries:
                cursor.execute("""
                    INSERT OR REPLACE INTO conversation_summaries_fts(rowid, summary, keywords)
                    VALUES (?, ?, ?)
                """, (summary_id, summary, keywords or ""))
            
            conn.commit()
            logger.info(f"Populated FTS table with {len(summaries)} existing summaries")
        except sqlite3.OperationalError as e:
            logger.warning(f"Could not populate FTS table: {e}")
        
        conn.close()
        return True
        
    except Exception as e:
        logger.error(f"Database migration failed: {e}")
        return False

if __name__ == "__main__":
    success = run_migration()
    if success:
        print("Migration completed successfully!")
    else:
        print("Migration failed!")
        exit(1)
