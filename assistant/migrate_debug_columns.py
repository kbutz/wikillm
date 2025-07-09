"""
Database Migration Script for Debug Columns
"""
import logging
from sqlalchemy import text
from database import get_db_session

logger = logging.getLogger(__name__)


def migrate_debug_columns():
    """Add debug columns to messages table"""
    logger.info("Starting database migration for debug columns...")
    
    try:
        with get_db_session() as db:
            # Add debug columns to messages table
            logger.info("Adding debug columns to messages table...")
            
            # Add debug_enabled column
            try:
                db.execute(text("ALTER TABLE messages ADD COLUMN debug_enabled BOOLEAN DEFAULT FALSE"))
                logger.info("Added debug_enabled column to messages table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add debug_enabled column: {e}")
            
            # Add debug_data column
            try:
                db.execute(text("ALTER TABLE messages ADD COLUMN debug_data JSON"))
                logger.info("Added debug_data column to messages table")
            except Exception as e:
                if "duplicate column name" not in str(e).lower():
                    logger.warning(f"Could not add debug_data column: {e}")
            
            db.commit()
            logger.info("Debug columns migration completed successfully")
            
    except Exception as e:
        logger.error(f"Debug columns migration failed: {e}")
        raise


if __name__ == "__main__":
    # Setup logging
    logging.basicConfig(level=logging.INFO)
    
    # Run migration
    migrate_debug_columns()
    
    print("Debug columns migration completed!")