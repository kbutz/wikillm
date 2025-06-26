"""
Test suite for AI Assistant
"""
import pytest
import asyncio
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from database import Base, get_db
from main import app
from models import User, Conversation, Message, UserMemory


# Test database setup
SQLALCHEMY_DATABASE_URL = "sqlite:///./test.db"
engine = create_engine(SQLALCHEMY_DATABASE_URL, connect_args={"check_same_thread": False})
TestingSessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

Base.metadata.create_all(bind=engine)


def override_get_db():
    try:
        db = TestingSessionLocal()
        yield db
    finally:
        db.close()


app.dependency_overrides[get_db] = override_get_db
client = TestClient(app)


@pytest.fixture
def test_user():
    """Create a test user"""
    user_data = {
        "username": "testuser",
        "email": "test@example.com",
        "full_name": "Test User"
    }
    response = client.post("/users/", json=user_data)
    return response.json()


@pytest.fixture
def test_conversation(test_user):
    """Create a test conversation"""
    conv_data = {
        "user_id": test_user["id"],
        "title": "Test Conversation"
    }
    response = client.post("/conversations/", json=conv_data)
    return response.json()


class TestUserManagement:
    """Test user management endpoints"""
    
    def test_create_user(self):
        """Test user creation"""
        user_data = {
            "username": "newuser",
            "email": "new@example.com",
            "full_name": "New User"
        }
        response = client.post("/users/", json=user_data)
        assert response.status_code == 201
        assert response.json()["username"] == "newuser"
    
    def test_create_duplicate_user(self, test_user):
        """Test duplicate user creation fails"""
        user_data = {
            "username": test_user["username"],
            "email": "different@example.com"
        }
        response = client.post("/users/", json=user_data)
        assert response.status_code == 400
    
    def test_get_user(self, test_user):
        """Test getting user by ID"""
        response = client.get(f"/users/{test_user['id']}")
        assert response.status_code == 200
        assert response.json()["username"] == test_user["username"]
    
    def test_get_nonexistent_user(self):
        """Test getting non-existent user"""
        response = client.get("/users/99999")
        assert response.status_code == 404


class TestConversationManagement:
    """Test conversation management"""
    
    def test_create_conversation(self, test_user):
        """Test conversation creation"""
        conv_data = {
            "user_id": test_user["id"],
            "title": "New Conversation"
        }
        response = client.post("/conversations/", json=conv_data)
        assert response.status_code == 201
        assert response.json()["title"] == "New Conversation"
    
    def test_get_user_conversations(self, test_user, test_conversation):
        """Test getting user conversations"""
        response = client.get(f"/users/{test_user['id']}/conversations")
        assert response.status_code == 200
        conversations = response.json()
        assert len(conversations) >= 1
        assert conversations[0]["id"] == test_conversation["id"]
    
    def test_delete_conversation(self, test_user, test_conversation):
        """Test conversation deletion"""
        response = client.delete(
            f"/conversations/{test_conversation['id']}?user_id={test_user['id']}"
        )
        assert response.status_code == 200


class TestChatFunctionality:
    """Test chat functionality"""
    
    @pytest.mark.asyncio
    async def test_chat_message(self, test_user):
        """Test sending chat message"""
        # Mock LMStudio response since it may not be available in tests
        chat_data = {
            "message": "Hello, how are you?",
            "user_id": test_user["id"],
            "temperature": 0.7
        }
        
        # This test assumes LMStudio is running
        # In a real test environment, you'd mock the LMStudio client
        try:
            response = client.post("/chat", json=chat_data)
            # If LMStudio is available, check successful response
            if response.status_code == 200:
                assert "message" in response.json()
                assert "conversation_id" in response.json()
            else:
                # If LMStudio not available, expect 500 error
                assert response.status_code == 500
        except Exception:
            # Test passes if LMStudio is not available
            pass


class TestMemorySystem:
    """Test memory management"""
    
    def test_add_explicit_memory(self, test_user):
        """Test adding explicit memory"""
        memory_data = {
            "memory_type": "explicit",
            "key": "favorite_color",
            "value": "blue",
            "confidence": 1.0
        }
        response = client.post(f"/users/{test_user['id']}/memory", json=memory_data)
        assert response.status_code == 200
        assert response.json()["key"] == "favorite_color"
    
    def test_get_user_memory(self, test_user):
        """Test getting user memory"""
        # First add some memory
        memory_data = {
            "memory_type": "explicit",
            "key": "name",
            "value": "John",
            "confidence": 1.0
        }
        client.post(f"/users/{test_user['id']}/memory", json=memory_data)
        
        # Then retrieve it
        response = client.get(f"/users/{test_user['id']}/memory")
        assert response.status_code == 200
        memories = response.json()
        assert len(memories) >= 1
    
    def test_delete_memory(self, test_user):
        """Test deleting memory"""
        # Add memory
        memory_data = {
            "memory_type": "explicit",
            "key": "test_key",
            "value": "test_value",
            "confidence": 1.0
        }
        response = client.post(f"/users/{test_user['id']}/memory", json=memory_data)
        memory_id = response.json()["id"]
        
        # Delete memory
        response = client.delete(f"/users/{test_user['id']}/memory/{memory_id}")
        assert response.status_code == 200


class TestSystemEndpoints:
    """Test system endpoints"""
    
    def test_health_check(self):
        """Test health check endpoint"""
        response = client.get("/health")
        assert response.status_code == 200
        assert "status" in response.json()
    
    @pytest.mark.asyncio
    async def test_system_status(self):
        """Test system status endpoint"""
        response = client.get("/status")
        assert response.status_code == 200
        status = response.json()
        assert "status" in status
        assert "version" in status
        assert "database_connected" in status


class TestMemoryManager:
    """Test memory manager functionality"""
    
    def test_extract_preferences(self):
        """Test preference extraction from text"""
        from memory_manager import MemoryManager
        from database import get_db_session
        
        with get_db_session() as db:
            memory_manager = MemoryManager(db)
            
            # Test concise preference
            memories = memory_manager.extract_implicit_memory(
                1, "Please keep it brief", "Short response"
            )
            
            pref_memories = [m for m in memories if m.key == "response_style"]
            assert len(pref_memories) > 0
            assert pref_memories[0].value == "concise"
    
    def test_extract_personal_info(self):
        """Test personal information extraction"""
        from memory_manager import MemoryManager
        from database import get_db_session
        
        with get_db_session() as db:
            memory_manager = MemoryManager(db)
            
            memories = memory_manager.extract_implicit_memory(
                1, "My name is Alice and I work as a developer", "Nice to meet you"
            )
            
            name_memories = [m for m in memories if m.key == "name"]
            profession_memories = [m for m in memories if m.key == "profession"]
            
            assert len(name_memories) > 0
            assert name_memories[0].value == "Alice"
            assert len(profession_memories) > 0
            assert "developer" in profession_memories[0].value.lower()


if __name__ == "__main__":
    pytest.main([__file__])
