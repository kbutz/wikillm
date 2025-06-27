#!/usr/bin/env python3
"""
Test script for enhanced memory functionality
"""
import asyncio
import httpx
import json
import time

# API configuration
BASE_URL = "http://localhost:8000"
USER_ID = 1

async def create_user_if_needed():
    """Create test user if it doesn't exist"""
    async with httpx.AsyncClient() as client:
        try:
            # Check if user exists
            response = await client.get(f"{BASE_URL}/users/{USER_ID}")
            if response.status_code == 200:
                print(f"User {USER_ID} already exists")
                return True
        except:
            pass
        
        # Create user
        try:
            response = await client.post(
                f"{BASE_URL}/users/",
                json={"username": "testuser", "email": "test@example.com"}
            )
            if response.status_code == 201:
                print("Created test user")
                return True
        except Exception as e:
            print(f"Error creating user: {e}")
            return False

async def test_pet_memory():
    """Test pet memory extraction and recall"""
    async with httpx.AsyncClient(timeout=30.0) as client:
        print("\n=== Testing Pet Memory ===")
        
        # First conversation: Tell about pet
        print("\n1. Creating first conversation about pet...")
        response = await client.post(
            f"{BASE_URL}/chat",
            json={
                "user_id": USER_ID,
                "message": "My dog's name is Crosby. He's a golden retriever and loves to play fetch."
            }
        )
        
        if response.status_code == 200:
            result = response.json()
            print(f"Assistant response: {result['message']['content'][:200]}...")
            conv1_id = result['conversation_id']
        else:
            print(f"Error: {response.status_code} - {response.text}")
            return
        
        # Wait for memory extraction
        print("\n2. Waiting for memory extraction...")
        await asyncio.sleep(3)
        
        # Check memories
        print("\n3. Checking stored memories...")
        response = await client.get(f"{BASE_URL}/users/{USER_ID}/memory")
        if response.status_code == 200:
            memories = response.json()
            print(f"Found {len(memories)} memories:")
            for mem in memories:
                if 'dog' in mem['key'] or 'pet' in mem['key']:
                    print(f"  - {mem['key']}: {mem['value']} (confidence: {mem['confidence']})")
        
        # Second conversation: Ask about pet
        print("\n4. Creating second conversation to test recall...")
        response = await client.post(
            f"{BASE_URL}/chat",
            json={
                "user_id": USER_ID,
                "message": "What do you know about my dog?"
            }
        )
        
        if response.status_code == 200:
            result = response.json()
            content = result['message']['content']
            print(f"\nAssistant response: {content}")
            
            # Check if Crosby is mentioned
            if "Crosby" in content:
                print("\n✅ SUCCESS: Assistant remembered the dog's name!")
            else:
                print("\n❌ FAILED: Assistant did not recall the dog's name")
        else:
            print(f"Error: {response.status_code} - {response.text}")
        
        # Test semantic memory search
        print("\n5. Testing semantic memory search...")
        response = await client.get(
            f"{BASE_URL}/users/{USER_ID}/memory/search",
            params={"q": "dog", "limit": 5}
        )
        
        if response.status_code == 200:
            results = response.json()
            print(f"Search results for 'dog': {results['total_found']} found")
            for mem in results['results']:
                print(f"  - {mem['key']}: {mem['value']}")

async def test_conversation_search():
    """Test cross-conversation search"""
    async with httpx.AsyncClient(timeout=30.0) as client:
        print("\n\n=== Testing Conversation Search ===")
        
        # Wait for summarization
        print("Waiting for conversation summarization...")
        await asyncio.sleep(2)
        
        # Search conversations
        response = await client.get(
            f"{BASE_URL}/users/{USER_ID}/conversations/search",
            params={"q": "dog Crosby", "limit": 5}
        )
        
        if response.status_code == 200:
            results = response.json()
            print(f"Found {results['total_found']} conversations about dogs:")
            for conv in results['results']:
                print(f"  - {conv['title']}: {conv['summary'][:100]}...")

async def main():
    """Run all tests"""
    print("Starting Enhanced Memory Tests")
    print("==============================")
    
    # Ensure user exists
    if not await create_user_if_needed():
        print("Failed to create user, exiting")
        return
    
    # Run tests
    await test_pet_memory()
    await test_conversation_search()
    
    print("\n\nTests completed!")

if __name__ == "__main__":
    asyncio.run(main())
