"""
Database migration script for debug persistence features
"""
import logging
from sqlalchemy import create_engine, MetaData, Table, Column, Integer, String, DateTime, Text, Float, Boolean, ForeignKey, JSON, inspect
from sqlalchemy.sql import func
from config import settings
from database import engine

logger = logging.getLogger(__name__)

def migrate_debug_persistence():
    """Add debug persistence tables to existing database"""
    
    metadata = MetaData()
    
    # Check if tables already exist
    inspector = inspect(engine)
    existing_tables = inspector.get_table_names()
    
    tables_to_create = []
    
    # Debug Sessions table
    if 'debug_sessions' not in existing_tables:
        debug_sessions = Table(
            'debug_sessions',
            metadata,
            Column('id', Integer, primary_key=True, index=True),
            Column('conversation_id', Integer, ForeignKey('conversations.id'), nullable=False),
            Column('user_id', Integer, ForeignKey('users.id'), nullable=False),
            Column('session_id', String(100), nullable=False, index=True),
            Column('started_at', DateTime, default=func.now()),
            Column('ended_at', DateTime, nullable=True),
            Column('is_active', Boolean, default=True),
            Column('total_messages', Integer, default=0),
            Column('total_steps', Integer, default=0),
            Column('total_tools_used', Integer, default=0),
            Column('total_processing_time', Float, default=0.0),
        )
        tables_to_create.append(debug_sessions)
    
    # Debug Steps table
    if 'debug_steps' not in existing_tables:
        debug_steps = Table(
            'debug_steps',
            metadata,
            Column('id', Integer, primary_key=True, index=True),
            Column('message_id', Integer, ForeignKey('messages.id'), nullable=False),
            Column('debug_session_id', Integer, ForeignKey('debug_sessions.id'), nullable=False),
            Column('step_id', String(100), nullable=False, index=True),
            Column('step_type', String(50), nullable=False),
            Column('step_order', Integer, nullable=False),
            Column('title', String(200), nullable=False),
            Column('description', Text, nullable=True),
            Column('timestamp', DateTime, default=func.now()),
            Column('duration_ms', Integer, nullable=True),
            Column('success', Boolean, default=True),
            Column('error_message', Text, nullable=True),
            Column('input_data', JSON, nullable=True),
            Column('output_data', JSON, nullable=True),
            Column('metadata', JSON, nullable=True),
        )
        tables_to_create.append(debug_steps)
    
    # LLM Requests table
    if 'llm_requests' not in existing_tables:
        llm_requests = Table(
            'llm_requests',
            metadata,
            Column('id', Integer, primary_key=True, index=True),
            Column('message_id', Integer, ForeignKey('messages.id'), nullable=False),
            Column('request_id', String(100), nullable=False, index=True),
            Column('model', String(100), nullable=False),
            Column('temperature', Float, nullable=True),
            Column('max_tokens', Integer, nullable=True),
            Column('stream', Boolean, default=False),
            Column('request_messages', JSON, nullable=False),
            Column('response_data', JSON, nullable=False),
            Column('timestamp', DateTime, default=func.now()),
            Column('processing_time_ms', Integer, nullable=True),
            Column('token_usage', JSON, nullable=True),
            Column('tools_available', JSON, nullable=True),
            Column('tools_used', JSON, nullable=True),
            Column('tool_calls', JSON, nullable=True),
            Column('tool_results', JSON, nullable=True),
        )
        tables_to_create.append(llm_requests)
    
    # Add new columns to existing messages table
    try:
        inspector = inspect(engine)
        existing_columns = [col['name'] for col in inspector.get_columns('messages')]
        
        with engine.connect() as conn:
            if 'debug_enabled' not in existing_columns:
                conn.execute('ALTER TABLE messages ADD COLUMN debug_enabled BOOLEAN DEFAULT FALSE')
                logger.info("Added debug_enabled column to messages table")
            
            if 'debug_data' not in existing_columns:
                conn.execute('ALTER TABLE messages ADD COLUMN debug_data JSON')
                logger.info("Added debug_data column to messages table")
            
            conn.commit()
    except Exception as e:
        logger.error(f"Failed to add columns to messages table: {e}")
    
    # Create new tables
    if tables_to_create:
        try:
            metadata.create_all(bind=engine, tables=tables_to_create)
            logger.info(f"Created {len(tables_to_create)} debug persistence tables")
        except Exception as e:
            logger.error(f"Failed to create debug persistence tables: {e}")
            raise
    else:
        logger.info("All debug persistence tables already exist")

def rollback_debug_persistence():
    """Remove debug persistence tables (for testing purposes)"""
    metadata = MetaData()
    metadata.reflect(bind=engine)
    
    tables_to_drop = ['debug_steps', 'llm_requests', 'debug_sessions']
    
    for table_name in tables_to_drop:
        if table_name in metadata.tables:
            try:
                metadata.tables[table_name].drop(engine)
                logger.info(f"Dropped table {table_name}")
            except Exception as e:
                logger.error(f"Failed to drop table {table_name}: {e}")
    
    # Remove columns from messages table
    try:
        with engine.connect() as conn:
            conn.execute('ALTER TABLE messages DROP COLUMN debug_enabled')
            conn.execute('ALTER TABLE messages DROP COLUMN debug_data')
            conn.commit()
            logger.info("Removed debug columns from messages table")
    except Exception as e:
        logger.error(f"Failed to remove columns from messages table: {e}")

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    migrate_debug_persistence()
    print("Database migration completed successfully!")
