"""
Enhanced Memory Management System with Entity and Relationship Extraction
"""
import logging
import re
import json
from typing import Dict, List, Optional, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc, func
from datetime import datetime, timedelta
from models import UserMemory, UserPreference, User, Message
from schemas import UserMemoryCreate, UserPreferenceCreate, MemoryType
from lmstudio_client import lmstudio_client
from llm_response_processor import LLMResponseProcessor

logger = logging.getLogger(__name__)


class EnhancedMemoryManager:
    """Enhanced memory management with entity extraction and semantic search"""

    def __init__(self, db: Session):
        self.db = db
        self.original_manager = MemoryManager(db)  # Keep original functionality
        self.response_processor = LLMResponseProcessor()

    async def extract_and_store_facts(
        self, 
        user_id: int, 
        user_message: str, 
        assistant_response: str,
        conversation_id: int
    ) -> List[UserMemory]:
        """Extract facts and entities from conversation using LLM"""

        # First, use original extraction
        memories = self.original_manager.extract_implicit_memory(user_id, user_message, assistant_response)

        # Then add advanced entity extraction
        try:
            extracted_facts = await self._extract_facts_with_llm(user_message, assistant_response)

            for fact in extracted_facts:
                # Check if this is a high-confidence fact worth storing
                if fact.get("confidence", 0) >= 0.6:
                    memory = UserMemoryCreate(
                        user_id=user_id,
                        memory_type=MemoryType.EXPLICIT if fact.get("confidence", 0) >= 0.8 else MemoryType.IMPLICIT,
                        key=fact["key"],
                        value=fact["value"],
                        confidence=fact["confidence"],
                        source=f"conversation_{conversation_id}"
                    )
                    memories.append(memory)

        except Exception as e:
            logger.error(f"LLM fact extraction failed: {e}")

        # Store all memories
        if memories:
            stored = self.original_manager.store_memories(memories)
            logger.info(f"Stored {len(stored)} facts for user {user_id}")
            return stored

        return []

    async def _extract_facts_with_llm(self, user_message: str, assistant_response: str) -> List[Dict[str, Any]]:
        """Use LLM to extract facts and relationships with enhanced validation"""
        
        # Clean assistant response to remove thinking tags
        cleaned_assistant_response = self.response_processor.process_memory_extraction_text(assistant_response)
        
        extraction_prompt = f"""Extract important facts and relationships from this conversation exchange.
Focus on:
1. Personal information (names, relationships, pets, family)
2. Preferences and opinions
3. Important dates or events
4. Skills or interests
5. Goals or plans

User said: {user_message}
Assistant responded: {cleaned_assistant_response}

Return a JSON array of facts. Each fact MUST have:
- key: a searchable identifier (e.g., "pet_dog_name", "favorite_food", "skill_programming")
- value: the fact itself as a STRING (e.g., "Crosby", "pizza", "Python expert")
- confidence: a number between 0.0 and 1.0

IMPORTANT: The "value" field must ALWAYS be a string. Convert numbers, booleans, etc. to strings.

Example format:
[
  {{"key": "pet_dog_name", "value": "Crosby", "confidence": 0.95}},
  {{"key": "dog_breed", "value": "Golden Retriever", "confidence": 0.8}},
  {{"key": "user_age", "value": "25", "confidence": 0.9}},
  {{"key": "likes_coffee", "value": "true", "confidence": 0.7}}
]

Extract only facts explicitly stated or strongly implied. Return only the JSON array, no other text."""

        try:
            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are a fact extraction system. Extract only clearly stated facts. Always ensure the 'value' field is a string."},
                    {"role": "user", "content": extraction_prompt}
                ],
                temperature=0.1,  # Low temperature for consistent extraction
                max_tokens=5000
            )

            content = response["choices"][0]["message"]["content"].strip()

            # Extract JSON from response
            json_match = re.search(r'\[.*\]', content, re.DOTALL)
            if json_match:
                facts = json.loads(json_match.group())

                # Validate and clean facts
                validated_facts = []
                for fact in facts:
                    if self._validate_fact(fact):
                        validated_facts.append(fact)
                    else:
                        logger.warning(f"Invalid fact filtered out: {fact}")

                return validated_facts
            else:
                logger.warning(f"No valid JSON found in LLM response: {content}")
                logger.warning(f"No valid JSON found in LLM response: {response}")
                return []

        except Exception as e:
            logger.error(f"Failed to extract facts with LLM: {e}")
            return []

    def _validate_fact(self, fact: Dict[str, Any]) -> bool:
        """Validate a fact dictionary and fix common issues"""
        try:
            # Check required fields
            if not all(key in fact for key in ['key', 'value', 'confidence']):
                return False

            # Ensure key is a string
            if not isinstance(fact['key'], str) or not fact['key'].strip():
                return False

            # Ensure confidence is a number between 0 and 1
            if not isinstance(fact['confidence'], (int, float)) or not 0 <= fact['confidence'] <= 1:
                return False

            # Convert value to string if it isn't already
            if not isinstance(fact['value'], str):
                fact['value'] = str(fact['value'])

            # Ensure value is not empty
            if not fact['value'].strip():
                return False

            return True

        except Exception as e:
            logger.warning(f"Fact validation failed: {e}")
            return False

    async def search_memories_semantic(
        self, 
        user_id: int, 
        query: str, 
        limit: int = 10
    ) -> List[UserMemory]:
        """Search memories using semantic understanding"""

        # First get all user memories
        all_memories = self.db.query(UserMemory).filter(
            UserMemory.user_id == user_id
        ).order_by(desc(UserMemory.confidence)).limit(100).all()

        if not all_memories:
            return []

        # Use LLM to rank memories by relevance to query
        try:
            memory_context = "\n".join([
                f"{i+1}. [{m.key}] {m.value}" 
                for i, m in enumerate(all_memories[:30])  # Limit for context
            ])

            ranking_prompt = f"""Given this query: "{query}"

And these memories:
{memory_context}

Return the numbers of the 5 most relevant memories as a comma-separated list.
For example: 3,7,1,15,9

Only return the numbers, nothing else."""

            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are a memory ranking system."},
                    {"role": "user", "content": ranking_prompt}
                ],
                temperature=0.1,
                max_tokens=50
            )

            numbers_str = response["choices"][0]["message"]["content"].strip()
            numbers = [int(n.strip()) - 1 for n in numbers_str.split(",") if n.strip().isdigit()]

            # Return ranked memories
            ranked_memories = []
            for idx in numbers[:limit]:
                if 0 <= idx < len(all_memories):
                    ranked_memories.append(all_memories[idx])

            return ranked_memories

        except Exception as e:
            logger.error(f"Semantic memory search failed: {e}")
            # Fallback to keyword search
            return self._fallback_keyword_search(all_memories, query, limit)

    def _fallback_keyword_search(
        self, 
        memories: List[UserMemory], 
        query: str, 
        limit: int
    ) -> List[UserMemory]:
        """Fallback keyword-based search"""
        query_lower = query.lower()
        scored_memories = []

        for memory in memories:
            score = 0
            key_lower = memory.key.lower()
            value_lower = memory.value.lower()

            # Exact matches get highest score
            if query_lower in value_lower:
                score += 10
            if query_lower in key_lower:
                score += 5

            # Partial word matches
            query_words = query_lower.split()
            for word in query_words:
                if len(word) > 2:  # Skip very short words
                    if word in value_lower:
                        score += 2
                    if word in key_lower:
                        score += 1

            if score > 0:
                scored_memories.append((score, memory))

        # Sort by score and return top results
        scored_memories.sort(key=lambda x: x[0], reverse=True)
        return [memory for score, memory in scored_memories[:limit]]

    async def get_contextual_memories(
        self, 
        user_id: int, 
        current_message: str,
        limit: int = 5
    ) -> str:
        """Get memories relevant to current message"""

        relevant_memories = await self.search_memories_semantic(user_id, current_message, limit)

        if not relevant_memories:
            # Try to get most recent/important memories
            relevant_memories = self.db.query(UserMemory).filter(
                UserMemory.user_id == user_id
            ).order_by(
                desc(UserMemory.confidence),
                desc(UserMemory.last_accessed)
            ).limit(limit).all()

        if not relevant_memories:
            return ""

        # Update access tracking
        for memory in relevant_memories:
            memory.last_accessed = datetime.now()
            memory.access_count += 1
        self.db.commit()

        # Format memories for context
        memory_parts = []
        for memory in relevant_memories:
            if memory.confidence >= 0.7:  # Only include confident memories
                memory_parts.append(f"- {memory.key}: {memory.value}")

        if memory_parts:
            return "Relevant information from memory:\n" + "\n".join(memory_parts)

        return ""


# Keep original MemoryManager class for backward compatibility
class MemoryManager:
    """Original Memory Management System"""

    def __init__(self, db: Session):
        self.db = db

    def extract_implicit_memory(self, user_id: int, message: str, response: str) -> List[UserMemoryCreate]:
        """Extract implicit memories from user interactions"""
        memories = []

        # Extract preferences from conversation patterns
        preferences = self._extract_preferences(message, response)
        for pref in preferences:
            memories.append(UserMemoryCreate(
                user_id=user_id,
                memory_type=MemoryType.IMPLICIT,
                key=pref["key"],
                value=pref["value"],
                confidence=pref["confidence"],
                source="conversation_analysis"
            ))

        # Extract personal information
        personal_info = self._extract_personal_info(message)
        for info in personal_info:
            memories.append(UserMemoryCreate(
                user_id=user_id,
                memory_type=MemoryType.IMPLICIT,
                key=info["key"],
                value=info["value"],
                confidence=info["confidence"],
                source="user_disclosure"
            ))

        return memories

    def _extract_preferences(self, message: str, response: str) -> List[Dict[str, Any]]:
        """Extract user preferences from conversation patterns"""
        preferences = []

        # Communication style preferences
        if any(phrase in message.lower() for phrase in ["brief", "short", "concise", "quick"]):
            preferences.append({
                "key": "response_style",
                "value": "concise",
                "confidence": 0.7
            })

        if any(phrase in message.lower() for phrase in ["detailed", "thorough", "comprehensive", "in-depth"]):
            preferences.append({
                "key": "response_style",
                "value": "detailed",
                "confidence": 0.7
            })

        # Technical level preferences
        if any(phrase in message.lower() for phrase in ["simple", "beginner", "new to", "don't understand"]):
            preferences.append({
                "key": "technical_level",
                "value": "beginner",
                "confidence": 0.6
            })

        if any(phrase in message.lower() for phrase in ["advanced", "expert", "professional", "technical"]):
            preferences.append({
                "key": "technical_level",
                "value": "advanced",
                "confidence": 0.6
            })

        return preferences

    def _extract_personal_info(self, message: str) -> List[Dict[str, Any]]:
        """Extract personal information from user messages"""
        personal_info = []

        # Enhanced patterns for pets and relationships
        pet_patterns = [
            r"(?:my|I have a?) (\w+)'s name is ([A-Z][a-z]+)",
            r"(?:my|I have a?) (\w+) (?:is )?(?:named|called) ([A-Z][a-z]+)",
            r"([A-Z][a-z]+) is my (\w+)",
            r"(?:my|I have a?) (\w+) ([A-Z][a-z]+)"  # "my dog Crosby"
        ]

        for pattern in pet_patterns:
            matches = re.finditer(pattern, message, re.IGNORECASE)
            for match in matches:
                if len(match.groups()) == 2:
                    # Determine which group is the pet type and which is the name
                    group1, group2 = match.groups()

                    # Common pet types
                    pet_types = ['dog', 'cat', 'bird', 'fish', 'hamster', 'rabbit', 'turtle', 'pet']

                    if group1.lower() in pet_types:
                        pet_type = group1.lower()
                        pet_name = group2
                    elif group2.lower() in pet_types:
                        pet_type = group2.lower()
                        pet_name = group1
                    else:
                        continue

                    personal_info.append({
                        "key": f"pet_{pet_type}_name",
                        "value": pet_name,
                        "confidence": 0.9
                    })

                    # Also store as a general pet entry
                    personal_info.append({
                        "key": "has_pet",
                        "value": f"{pet_type} named {pet_name}",
                        "confidence": 0.9
                    })

        # Name extraction
        name_patterns = [
            r"(?:my name is|i'm|i am|call me) ([A-Z][a-z]+)",
            r"name[':]\s*([A-Z][a-z]+)"
        ]

        for pattern in name_patterns:
            match = re.search(pattern, message, re.IGNORECASE)
            if match:
                personal_info.append({
                    "key": "name",
                    "value": match.group(1),
                    "confidence": 0.9
                })

        # Family relationships
        family_patterns = [
            r"(?:my|I have \w+) (wife|husband|spouse|partner|mother|father|mom|dad|son|daughter|brother|sister) (?:is )?(?:named|called)? ([A-Z][a-z]+)",
            r"([A-Z][a-z]+) is my (wife|husband|spouse|partner|mother|father|mom|dad|son|daughter|brother|sister)"
        ]

        for pattern in family_patterns:
            matches = re.finditer(pattern, message, re.IGNORECASE)
            for match in matches:
                groups = match.groups()
                if len(groups) == 2:
                    # Determine order
                    if groups[1].lower() in ['wife', 'husband', 'spouse', 'partner', 'mother', 'father', 'mom', 'dad', 'son', 'daughter', 'brother', 'sister']:
                        relation = groups[1].lower()
                        name = groups[0]
                    else:
                        relation = groups[0].lower()
                        name = groups[1]

                    personal_info.append({
                        "key": f"family_{relation}_name",
                        "value": name,
                        "confidence": 0.85
                    })

        # Location extraction
        location_patterns = [
            r"(?:I|i) (?:live|reside|stay) in ([A-Za-z\s]+)",
            r"(?:I|i) am (?:from|in) ([A-Za-z\s]+)",
            r"(?:my|My) (?:home|location|city|town) is ([A-Za-z\s]+)"
        ]

        for pattern in location_patterns:
            match = re.search(pattern, message, re.IGNORECASE)
            if match:
                location = match.group(1).strip()
                personal_info.append({
                    "key": "location",
                    "value": location,
                    "confidence": 0.9
                })

        # Hobbies and interests
        hobby_patterns = [
            r"(?:I|i) (?:like|love|enjoy|am interested in) (\w+(?:\s+\w+){0,2})",
            r"(?:my|My) (?:hobby|hobbies|interest|interests) (?:is|are|include) (\w+(?:\s+\w+){0,2})"
        ]

        for pattern in hobby_patterns:
            matches = re.finditer(pattern, message, re.IGNORECASE)
            for match in matches:
                hobby = match.group(1).strip().lower()
                if len(hobby) > 2 and hobby not in ['to', 'the', 'a', 'an']:
                    personal_info.append({
                        "key": f"interest_{hobby.replace(' ', '_')}",
                        "value": hobby,
                        "confidence": 0.7
                    })

        return personal_info

    def store_memory(self, memory: UserMemoryCreate) -> UserMemory:
        """Store a single memory entry"""
        # Check if similar memory already exists
        existing = self.db.query(UserMemory).filter(
            and_(
                UserMemory.user_id == memory.user_id,
                UserMemory.key == memory.key,
                UserMemory.memory_type == memory.memory_type
            )
        ).first()

        if existing:
            # Update existing memory with higher confidence or newer information
            if memory.confidence >= existing.confidence:
                existing.value = memory.value
                existing.confidence = memory.confidence
                existing.updated_at = datetime.now()
                existing.access_count += 1
                self.db.commit()
                return existing
            else:
                return existing

        # Create new memory entry
        db_memory = UserMemory(**memory.dict())
        self.db.add(db_memory)
        self.db.commit()
        self.db.refresh(db_memory)

        logger.info(f"Stored new memory: {memory.key} for user {memory.user_id}")
        return db_memory

    def store_memories(self, memories: List[UserMemoryCreate]) -> List[UserMemory]:
        """Store multiple memory entries"""
        stored_memories = []
        for memory in memories:
            stored_memory = self.store_memory(memory)
            stored_memories.append(stored_memory)
        return stored_memories

    def get_user_memories(
        self,
        user_id: int,
        memory_type: Optional[MemoryType] = None,
        key_pattern: Optional[str] = None,
        limit: int = 100
    ) -> List[UserMemory]:
        """Retrieve user memories with optional filtering"""
        query = self.db.query(UserMemory).filter(UserMemory.user_id == user_id)

        if memory_type:
            query = query.filter(UserMemory.memory_type == memory_type)

        if key_pattern:
            query = query.filter(UserMemory.key.like(f"%{key_pattern}%"))

        return query.order_by(desc(UserMemory.last_accessed)).limit(limit).all()

    def get_memory_context(self, user_id: int) -> str:
        """Generate context string from user memories for AI prompting"""
        memories = self.get_user_memories(user_id, limit=20)

        if not memories:
            return ""

        context_parts = []

        # Group memories by type
        explicit_memories = [m for m in memories if m.memory_type == MemoryType.EXPLICIT]
        implicit_memories = [m for m in memories if m.memory_type == MemoryType.IMPLICIT]

        if explicit_memories:
            context_parts.append("User has explicitly mentioned:")
            for memory in explicit_memories[:5]:  # Top 5 explicit
                context_parts.append(f"- {memory.key}: {memory.value}")

        if implicit_memories:
            context_parts.append("\nBased on conversation patterns:")
            for memory in implicit_memories[:10]:  # Top 10 implicit
                if memory.confidence > 0.5:  # Only include confident predictions
                    context_parts.append(f"- {memory.key}: {memory.value}")

        return "\n".join(context_parts)

    def update_memory_access(self, memory_id: int):
        """Update memory access tracking"""
        memory = self.db.query(UserMemory).filter(UserMemory.id == memory_id).first()
        if memory:
            memory.last_accessed = datetime.now()
            memory.access_count += 1
            self.db.commit()

    def consolidate_memories(self, user_id: int):
        """Consolidate similar memories to reduce redundancy"""
        memories = self.get_user_memories(user_id)

        # Group by key and memory type
        memory_groups = {}
        for memory in memories:
            key = (memory.key, memory.memory_type)
            if key not in memory_groups:
                memory_groups[key] = []
            memory_groups[key].append(memory)

        # Consolidate groups with multiple entries
        for (key, memory_type), group in memory_groups.items():
            if len(group) > 1:
                # Keep the most confident and recent memory
                best_memory = max(group, key=lambda m: (m.confidence, m.updated_at))

                # Remove others
                for memory in group:
                    if memory.id != best_memory.id:
                        self.db.delete(memory)

                logger.info(f"Consolidated {len(group)} memories for key: {key}")

        self.db.commit()

    def get_user_preferences(self, user_id: int) -> Dict[str, Any]:
        """Get user preferences as a dictionary"""
        preferences = self.db.query(UserPreference).filter(
            UserPreference.user_id == user_id
        ).all()

        pref_dict = {}
        for pref in preferences:
            if pref.category not in pref_dict:
                pref_dict[pref.category] = {}
            pref_dict[pref.category][pref.key] = pref.value

        return pref_dict

    def set_user_preference(self, user_id: int, category: str, key: str, value: Any) -> UserPreference:
        """Set a user preference"""
        existing = self.db.query(UserPreference).filter(
            and_(
                UserPreference.user_id == user_id,
                UserPreference.category == category,
                UserPreference.key == key
            )
        ).first()

        if existing:
            existing.value = value
            existing.updated_at = datetime.now()
            self.db.commit()
            return existing

        preference = UserPreference(
            user_id=user_id,
            category=category,
            key=key,
            value=value
        )
        self.db.add(preference)
        self.db.commit()
        self.db.refresh(preference)

        return preference
