"""
Development utilities and scripts
"""
import asyncio
import json
from datetime import datetime, timedelta
from database import get_db_session
from models import User, Conversation, Message, UserMemory
from memory_manager import MemoryManager
from conversation_manager import ConversationManager


def create_sample_data():
    """Create sample data for development/testing"""
    with get_db_session() as db:
        # Create sample user
        user = User(
            username="alice_dev",
            email="alice@example.com",
            full_name="Alice Developer"
        )
        db.add(user)
        db.commit()
        db.refresh(user)
        
        # Create sample conversations
        conv_manager = ConversationManager(db)
        memory_manager = MemoryManager(db)
        
        conv1 = conv_manager.create_conversation(user.id, "Python Learning")
        conv2 = conv_manager.create_conversation(user.id, "Project Planning")
        
        # Add sample messages
        messages = [
            ("Hi, I'm new to Python programming", "user"),
            ("Welcome! I'd be happy to help you learn Python. What would you like to start with?", "assistant"),
            ("Can you explain variables?", "user"),
            ("Variables in Python are containers for storing data values. Unlike many programming languages, Python has no command for declaring a variable. You create one by assigning a value to it.", "assistant"),
        ]
        
        for content, role in messages:
            conv_manager.add_message(conv1.id, role, content)
        
        # Add sample explicit memories
        explicit_memories = [
            ("name", "Alice", 1.0),
            ("profession", "Software Developer", 1.0),
            ("programming_language", "Python", 1.0),
            ("experience_level", "Beginner", 0.9),
        ]
        
        from schemas import UserMemoryCreate, MemoryType
        
        for key, value, confidence in explicit_memories:
            memory = UserMemoryCreate(
                user_id=user.id,
                memory_type=MemoryType.EXPLICIT,
                key=key,
                value=value,
                confidence=confidence,
                source="manual_entry"
            )
            memory_manager.store_memory(memory)
        
        # Add sample implicit memories
        implicit_memories = [
            ("response_style", "detailed", 0.8),
            ("technical_level", "beginner", 0.7),
            ("learning_style", "step_by_step", 0.9),
        ]
        
        for key, value, confidence in implicit_memories:
            memory = UserMemoryCreate(
                user_id=user.id,
                memory_type=MemoryType.IMPLICIT,
                key=key,
                value=value,
                confidence=confidence,
                source="conversation_analysis"
            )
            memory_manager.store_memory(memory)
        
        print(f"✅ Created sample user: {user.username} (ID: {user.id})")
        print(f"✅ Created {len([conv1, conv2])} conversations")
        print(f"✅ Added {len(messages)} messages")
        print(f"✅ Created {len(explicit_memories + implicit_memories)} memory entries")


def analyze_memory_patterns():
    """Analyze memory patterns in the database"""
    with get_db_session() as db:
        memories = db.query(UserMemory).all()
        
        # Group by memory type
        type_counts = {}
        confidence_stats = {}
        
        for memory in memories:
            # Count by type
            if memory.memory_type not in type_counts:
                type_counts[memory.memory_type] = 0
            type_counts[memory.memory_type] += 1
            
            # Confidence stats
            if memory.memory_type not in confidence_stats:
                confidence_stats[memory.memory_type] = []
            confidence_stats[memory.memory_type].append(memory.confidence)
        
        print("📊 Memory Analysis")
        print("=================")
        
        for mem_type, count in type_counts.items():
            avg_confidence = sum(confidence_stats[mem_type]) / len(confidence_stats[mem_type])
            print(f"{mem_type.capitalize()}: {count} entries, avg confidence: {avg_confidence:.2f}")
        
        # Most common keys
        key_counts = {}
        for memory in memories:
            if memory.key not in key_counts:
                key_counts[memory.key] = 0
            key_counts[memory.key] += 1
        
        print("\n🔑 Most Common Memory Keys:")
        sorted_keys = sorted(key_counts.items(), key=lambda x: x[1], reverse=True)
        for key, count in sorted_keys[:10]:
            print(f"  {key}: {count}")


def simulate_conversation():
    """Simulate a conversation to test memory extraction"""
    with get_db_session() as db:
        user = db.query(User).first()
        if not user:
            print("❌ No users found. Run create_sample_data() first.")
            return
        
        conv_manager = ConversationManager(db)
        memory_manager = MemoryManager(db)
        
        # Create new conversation
        conversation = conv_manager.create_conversation(user.id, "Memory Test")
        
        # Simulate conversation exchanges
        exchanges = [
            ("I prefer short answers", "I'll keep my responses concise."),
            ("I work as a data scientist", "That's great! Data science is a fascinating field."),
            ("I'm working on a machine learning project", "Excellent! What type of ML problem are you solving?"),
            ("Break it down step by step please", "I'll walk you through each step systematically."),
        ]
        
        for user_msg, assistant_msg in exchanges:
            # Add user message
            conv_manager.add_message(conversation.id, "user", user_msg)
            
            # Extract memories from the exchange
            memories = memory_manager.extract_implicit_memory(user.id, user_msg, assistant_msg)
            
            if memories:
                stored = memory_manager.store_memories(memories)
                print(f"💭 Extracted {len(stored)} memories from: '{user_msg[:50]}...'")
                for memory in stored:
                    print(f"   {memory.key}: {memory.value} (confidence: {memory.confidence})")
            
            # Add assistant message
            conv_manager.add_message(conversation.id, "assistant", assistant_msg)
        
        print(f"\n✅ Simulated conversation with {len(exchanges)} exchanges")


def export_user_data(user_id: int, filename: str = None):
    """Export user data to JSON file"""
    if not filename:
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"user_{user_id}_export_{timestamp}.json"
    
    with get_db_session() as db:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            print(f"❌ User {user_id} not found")
            return
        
        # Get user data
        conversations = db.query(Conversation).filter(
            Conversation.user_id == user_id,
            Conversation.is_active == True
        ).all()
        
        memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
        
        # Build export data
        export_data = {
            "user": {
                "id": user.id,
                "username": user.username,
                "email": user.email,
                "full_name": user.full_name,
                "created_at": user.created_at.isoformat(),
            },
            "conversations": [],
            "memories": []
        }
        
        # Add conversations with messages
        for conv in conversations:
            messages = db.query(Message).filter(Message.conversation_id == conv.id).all()
            
            conv_data = {
                "id": conv.id,
                "title": conv.title,
                "created_at": conv.created_at.isoformat(),
                "updated_at": conv.updated_at.isoformat(),
                "messages": [
                    {
                        "role": msg.role,
                        "content": msg.content,
                        "timestamp": msg.timestamp.isoformat(),
                        "processing_time": msg.processing_time
                    }
                    for msg in messages
                ]
            }
            export_data["conversations"].append(conv_data)
        
        # Add memories
        for memory in memories:
            memory_data = {
                "memory_type": memory.memory_type,
                "key": memory.key,
                "value": memory.value,
                "confidence": memory.confidence,
                "source": memory.source,
                "created_at": memory.created_at.isoformat(),
                "access_count": memory.access_count
            }
            export_data["memories"].append(memory_data)
        
        # Write to file
        with open(filename, 'w') as f:
            json.dump(export_data, f, indent=2)
        
        print(f"✅ Exported user data to {filename}")
        print(f"   User: {user.username}")
        print(f"   Conversations: {len(export_data['conversations'])}")
        print(f"   Memories: {len(export_data['memories'])}")


def cleanup_test_data():
    """Clean up test/development data"""
    with get_db_session() as db:
        # Delete test users (those with 'test' or 'dev' in username)
        test_users = db.query(User).filter(
            User.username.like('%test%') | User.username.like('%dev%')
        ).all()
        
        for user in test_users:
            # Delete associated data will cascade
            db.delete(user)
        
        db.commit()
        print(f"🧹 Cleaned up {len(test_users)} test users and associated data")


def memory_consolidation_task():
    """Run memory consolidation for all users"""
    with get_db_session() as db:
        users = db.query(User).all()
        memory_manager = MemoryManager(db)
        
        total_consolidated = 0
        for user in users:
            before_count = db.query(UserMemory).filter(UserMemory.user_id == user.id).count()
            memory_manager.consolidate_memories(user.id)
            after_count = db.query(UserMemory).filter(UserMemory.user_id == user.id).count()
            
            consolidated = before_count - after_count
            if consolidated > 0:
                print(f"👤 {user.username}: consolidated {consolidated} memories")
                total_consolidated += consolidated
        
        print(f"✅ Total memories consolidated: {total_consolidated}")


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 2:
        print("Available commands:")
        print("  sample_data    - Create sample data")
        print("  analyze        - Analyze memory patterns")
        print("  simulate       - Simulate conversation")
        print("  export <user_id> - Export user data")
        print("  cleanup        - Clean up test data")
        print("  consolidate    - Run memory consolidation")
        sys.exit(1)
    
    command = sys.argv[1]
    
    if command == "sample_data":
        create_sample_data()
    elif command == "analyze":
        analyze_memory_patterns()
    elif command == "simulate":
        simulate_conversation()
    elif command == "export" and len(sys.argv) > 2:
        export_user_data(int(sys.argv[2]))
    elif command == "cleanup":
        cleanup_test_data()
    elif command == "consolidate":
        memory_consolidation_task()
    else:
        print(f"Unknown command: {command}")
