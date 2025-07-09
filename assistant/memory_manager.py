"""
Enhanced Memory Management System with Consolidated User Profiles
"""
import logging
import re
import json
from typing import Dict, List, Optional, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc, func
from datetime import datetime, timedelta
from collections import defaultdict
from models import UserMemory, UserPreference, User, Message
from schemas import UserMemoryCreate, UserPreferenceCreate, MemoryType
from lmstudio_client import lmstudio_client
from llm_response_processor import LLMResponseProcessor

logger = logging.getLogger(__name__)


class EnhancedMemoryManager:
    """Enhanced memory management with consolidated user profiles and conflict resolution"""

    def __init__(self, db: Session):
        self.db = db
        self.original_manager = MemoryManager(db)
        self.response_processor = LLMResponseProcessor()

    async def get_consolidated_user_profile(self, user_id: int) -> Dict[str, Any]:
        """Get consolidated, deduplicated user profile organized by category"""
        try:
            # Get all high-confidence memories (≥0.7 for system prompts)
            memories = self.db.query(UserMemory).filter(
                and_(
                    UserMemory.user_id == user_id,
                    UserMemory.confidence >= 0.7
                )
            ).order_by(
                desc(UserMemory.confidence),
                desc(UserMemory.updated_at)
            ).all()

            if not memories:
                return {}

            # Initialize profile structure
            profile = {
                "personal": {},      # Names, relationships, pets
                "preferences": {},   # Likes, dislikes, communication style
                "skills": {},        # Technical abilities, expertise
                "projects": {},      # Current goals, projects
                "context": {}        # Working directory, recent topics
            }

            # Categorize and consolidate memories
            await self._categorize_memories(memories, profile)
            
            # Resolve conflicts and deduplicate
            await self._resolve_memory_conflicts(profile)
            
            # Filter out empty categories
            return {k: v for k, v in profile.items() if v}

        except Exception as e:
            logger.error(f"Failed to get consolidated user profile: {e}")
            return {}

    async def _categorize_memories(self, memories: List[UserMemory], profile: Dict[str, Any]):
        """Categorize memories into structured profile sections"""
        
        # Category patterns for automatic classification
        category_patterns = {
            "personal": [
                r"^name$", r"^pet_", r"^family_", r"^location$", r"^age$", r"^has_pet$"
            ],
            "preferences": [
                r"^response_style$", r"^technical_level$", r"^favorite_", r"^likes_", 
                r"^prefers_", r"^communication_"
            ],
            "skills": [
                r"^skill_", r"^expertise_", r"^programming_", r"^language_", r"^technology_"
            ],
            "projects": [
                r"^goal_", r"^project_", r"^working_on", r"^current_"
            ],
            "context": [
                r"^allowed_directory", r"^recent_", r"^last_", r"^current_topic"
            ]
        }

        for memory in memories:
            # Determine category
            category = self._determine_memory_category(memory.key, category_patterns)
            
            # Clean up the key for better presentation
            clean_key = self._clean_memory_key(memory.key)
            
            # Store in appropriate category
            if category and clean_key:
                profile[category][clean_key] = memory.value

    def _determine_memory_category(self, key: str, patterns: Dict[str, List[str]]) -> str:
        """Determine which category a memory key belongs to"""
        for category, pattern_list in patterns.items():
            for pattern in pattern_list:
                if re.match(pattern, key, re.IGNORECASE):
                    return category
        return "context"  # Default category

    def _clean_memory_key(self, key: str) -> str:
        """Clean memory key for better presentation"""
        # Remove common prefixes
        key = re.sub(r'^(pet_|family_|skill_|interest_|goal_|project_)', '', key)
        
        # Convert underscores to spaces and title case
        key = key.replace('_', ' ').title()
        
        # Handle special cases
        key = key.replace('Ai', 'AI').replace('Api', 'API').replace('Ui', 'UI')
        
        return key

    async def _resolve_memory_conflicts(self, profile: Dict[str, Any]):
        """Resolve conflicts and deduplicate similar memories"""
        
        # Define conflict resolution rules
        conflict_groups = {
            "pet_info": ["dog", "cat", "pet", "animal"],
            "name_variations": ["name", "username", "full name"],
            "location_info": ["location", "city", "home", "address"],
            "programming_langs": ["python", "javascript", "java", "programming"]
        }

        for category_name, category_data in profile.items():
            if not isinstance(category_data, dict):
                continue
                
            # Check for conflicts within each category
            for conflict_group, keywords in conflict_groups.items():
                conflicting_keys = []
                
                for key in category_data.keys():
                    if any(keyword in key.lower() for keyword in keywords):
                        conflicting_keys.append(key)
                
                # Resolve conflicts by keeping the most specific/recent
                if len(conflicting_keys) > 1:
                    await self._resolve_conflict_group(category_data, conflicting_keys)

    async def _resolve_conflict_group(self, category_data: Dict[str, str], conflicting_keys: List[str]):
        """Resolve conflicts within a group of related keys"""
        
        # Create consolidated entry
        consolidated_info = []
        for key in conflicting_keys:
            value = category_data[key]
            # Skip obviously incorrect or duplicate values
            if value and value.lower() not in ['true', 'false', 'unknown', 'undefined']:
                consolidated_info.append(f"{key}: {value}")
        
        if consolidated_info:
            # Use the first key as the primary key
            primary_key = conflicting_keys[0]
            if len(consolidated_info) == 1:
                # Single item, use just the value
                category_data[primary_key] = consolidated_info[0].split(': ', 1)[1]
            else:
                # Multiple items, create consolidated entry
                category_data[primary_key] = "; ".join(consolidated_info)
            
            # Remove other conflicting keys
            for key in conflicting_keys[1:]:
                category_data.pop(key, None)

    async def get_relevant_memories_for_query(
        self,
        user_id: int,
        query: str,
        limit: int = 5
    ) -> List[UserMemory]:
        """Get memories most relevant to a specific query"""
        
        # Get consolidated profile first
        profile = await self.get_consolidated_user_profile(user_id)
        
        if not profile:
            return []

        # Use semantic search on consolidated profile
        try:
            # Create searchable text from profile
            profile_text = self._flatten_profile_for_search(profile)
            
            # Use LLM to identify relevant sections
            relevance_prompt = f"""
            Given this user query: "{query}"
            
            And this user profile:
            {profile_text}
            
            Which profile information is most relevant to the query?
            Return the top 3-5 most relevant facts as a JSON array of strings.
            
            Example: ["name: John Doe", "skill: Python programming", "project: web scraper"]
            """
            
            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are a relevance assessment system. Return only JSON."},
                    {"role": "user", "content": relevance_prompt}
                ],
                temperature=0.1,
                max_tokens=200
            )
            
            content = response["choices"][0]["message"]["content"].strip()
            
            # Extract JSON array
            import json
            try:
                relevant_facts = json.loads(content)
                
                # Convert back to UserMemory objects for compatibility
                relevant_memories = []
                for fact in relevant_facts[:limit]:
                    # Create pseudo-memory objects for consistency
                    if ': ' in fact:
                        key, value = fact.split(': ', 1)
                        # Find original memory in database
                        original_memory = self.db.query(UserMemory).filter(
                            and_(
                                UserMemory.user_id == user_id,
                                UserMemory.value == value
                            )
                        ).first()
                        
                        if original_memory:
                            relevant_memories.append(original_memory)
                
                return relevant_memories
                
            except json.JSONDecodeError:
                logger.warning(f"Failed to parse relevance response: {content}")
                return []
                
        except Exception as e:
            logger.error(f"Relevance search failed: {e}")
            return []

    def _flatten_profile_for_search(self, profile: Dict[str, Any]) -> str:
        """Flatten profile into searchable text"""
        flattened = []
        
        for category, items in profile.items():
            if isinstance(items, dict):
                for key, value in items.items():
                    flattened.append(f"{key}: {value}")
            else:
                flattened.append(f"{category}: {items}")
        
        return "\n".join(flattened)

    async def extract_and_store_facts(
        self, 
        user_id: int, 
        user_message: str, 
        assistant_response: str,
        conversation_id: int
    ) -> List[UserMemory]:
        """Extract and store facts with improved deduplication"""

        # Use original extraction as baseline
        memories = self.original_manager.extract_implicit_memory(user_id, user_message, assistant_response)

        # Add advanced LLM-based extraction
        try:
            extracted_facts = await self._extract_facts_with_llm(user_message, assistant_response)

            for fact in extracted_facts:
                # Higher confidence threshold for storage
                if fact.get("confidence", 0) >= 0.7:
                    memory = UserMemoryCreate(
                        user_id=user_id,
                        memory_type=MemoryType.EXPLICIT if fact.get("confidence", 0) >= 0.85 else MemoryType.IMPLICIT,
                        key=fact["key"],
                        value=fact["value"],
                        confidence=fact["confidence"],
                        source=f"conversation_{conversation_id}"
                    )
                    memories.append(memory)

        except Exception as e:
            logger.error(f"LLM fact extraction failed: {e}")

        # Store and deduplicate
        if memories:
            stored = await self._store_and_deduplicate_memories(memories)
            logger.info(f"Stored {len(stored)} facts for user {user_id}")
            return stored

        return []

    async def _store_and_deduplicate_memories(self, memories: List[UserMemoryCreate]) -> List[UserMemory]:
        """Store memories with intelligent deduplication"""
        stored_memories = []
        
        for memory in memories:
            # Check for existing similar memories
            existing_memories = self.db.query(UserMemory).filter(
                and_(
                    UserMemory.user_id == memory.user_id,
                    UserMemory.key == memory.key
                )
            ).all()
            
            if existing_memories:
                # Update or merge with existing memory
                best_existing = max(existing_memories, key=lambda m: m.confidence)
                
                if memory.confidence > best_existing.confidence:
                    # Update existing memory
                    best_existing.value = memory.value
                    best_existing.confidence = memory.confidence
                    best_existing.updated_at = datetime.now()
                    best_existing.source = memory.source
                    
                    # Remove other duplicates
                    for existing in existing_memories:
                        if existing.id != best_existing.id:
                            self.db.delete(existing)
                    
                    stored_memories.append(best_existing)
                else:
                    # Keep existing memory, don't store new one
                    stored_memories.append(best_existing)
            else:
                # Store new memory
                db_memory = UserMemory(**memory.dict())
                self.db.add(db_memory)
                stored_memories.append(db_memory)
        
        self.db.commit()
        
        # Refresh stored memories
        for memory in stored_memories:
            self.db.refresh(memory)
            
        return stored_memories

    async def _extract_facts_with_llm(self, user_message: str, assistant_response: str) -> List[Dict[str, Any]]:
        """Enhanced fact extraction with better validation"""
        
        # Clean assistant response
        cleaned_response = self.response_processor.process_memory_extraction_text(assistant_response)
        
        extraction_prompt = f"""Extract clear, factual information from this conversation.

User: {user_message}
Assistant: {cleaned_response}

Focus on:
1. Concrete personal facts (names, locations, ages, etc.)
2. Clear preferences and opinions
3. Specific skills or expertise mentioned
4. Current projects or goals
5. Technical details (programming languages, tools, etc.)

Return a JSON array. Each fact must have:
- key: descriptive identifier (e.g., "pet_dog_name", "programming_language", "current_project")
- value: the actual fact as a string
- confidence: number 0.0-1.0 (only include facts with confidence ≥ 0.7)

Rules:
- Only extract explicitly stated information
- Avoid assumptions or inferences
- Keep values factual and concise
- Use consistent key naming (snake_case)

Example:
[
  {{"key": "pet_dog_name", "value": "Max", "confidence": 0.9}},
  {{"key": "programming_language", "value": "Python", "confidence": 0.8}},
  {{"key": "current_project", "value": "web scraper", "confidence": 0.85}}
]"""

        try:
            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are a precise fact extraction system. Extract only clear, factual information."},
                    {"role": "user", "content": extraction_prompt}
                ],
                temperature=0.1,
                max_tokens=1000
            )

            content = response["choices"][0]["message"]["content"].strip()

            # Extract and validate JSON
            json_match = re.search(r'\[.*\]', content, re.DOTALL)
            if json_match:
                facts = json.loads(json_match.group())
                return [fact for fact in facts if self._validate_extracted_fact(fact)]
            
            return []

        except Exception as e:
            logger.error(f"LLM fact extraction failed: {e}")
            return []

    def _validate_extracted_fact(self, fact: Dict[str, Any]) -> bool:
        """Validate extracted fact with enhanced rules"""
        try:
            # Check required fields
            if not all(key in fact for key in ['key', 'value', 'confidence']):
                return False

            # Validate key format
            key = fact['key']
            if not isinstance(key, str) or not key.strip():
                return False
            
            # Key should be descriptive (not too short)
            if len(key) < 3:
                return False

            # Validate confidence
            confidence = fact['confidence']
            if not isinstance(confidence, (int, float)) or not 0.7 <= confidence <= 1.0:
                return False

            # Validate value
            value = str(fact['value']).strip()
            if not value or value.lower() in ['true', 'false', 'unknown', 'undefined', 'null']:
                return False

            # Value should be meaningful (not too short unless it's a name)
            if len(value) < 2:
                return False

            return True

        except Exception as e:
            logger.warning(f"Fact validation failed: {e}")
            return False

    async def get_contextual_memories(
        self, 
        user_id: int, 
        current_message: str,
        limit: int = 5
    ) -> str:
        """Get contextual memories using consolidated profile"""
        
        # Get relevant memories for the current message
        relevant_memories = await self.get_relevant_memories_for_query(
            user_id, current_message, limit
        )

        if not relevant_memories:
            return ""

        # Update access tracking
        for memory in relevant_memories:
            memory.last_accessed = datetime.now()
            memory.access_count += 1
        self.db.commit()

        # Format for context
        memory_parts = []
        for memory in relevant_memories:
            clean_key = self._clean_memory_key(memory.key)
            memory_parts.append(f"- {clean_key}: {memory.value}")

        if memory_parts:
            return "Relevant context:\n" + "\n".join(memory_parts)

        return ""


# Enhanced original MemoryManager with improved methods
class MemoryManager:
    """Enhanced original Memory Management System"""

    def __init__(self, db: Session):
        self.db = db

    def extract_implicit_memory(self, user_id: int, message: str, response: str) -> List[UserMemoryCreate]:
        """Extract implicit memories with improved patterns"""
        memories = []

        # Extract preferences
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
        """Extract user preferences with improved detection"""
        preferences = []

        # Communication style
        if any(phrase in message.lower() for phrase in ["brief", "short", "concise", "quick"]):
            preferences.append({
                "key": "communication_style",
                "value": "concise",
                "confidence": 0.7
            })

        if any(phrase in message.lower() for phrase in ["detailed", "thorough", "comprehensive"]):
            preferences.append({
                "key": "communication_style",
                "value": "detailed",
                "confidence": 0.7
            })

        # Technical level
        if any(phrase in message.lower() for phrase in ["beginner", "new to", "don't understand"]):
            preferences.append({
                "key": "technical_level",
                "value": "beginner",
                "confidence": 0.6
            })

        if any(phrase in message.lower() for phrase in ["advanced", "expert", "professional"]):
            preferences.append({
                "key": "technical_level",
                "value": "advanced",
                "confidence": 0.6
            })

        return preferences

    def _extract_personal_info(self, message: str) -> List[Dict[str, Any]]:
        """Extract personal information with improved patterns"""
        personal_info = []

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

        # Pet information with improved accuracy
        pet_patterns = [
            r"(?:my|i have a?) (dog|cat|bird|fish|hamster|rabbit|turtle|pet) (?:is )?(?:named|called) ([A-Z][a-z]+)",
            r"([A-Z][a-z]+) is my (dog|cat|bird|fish|hamster|rabbit|turtle|pet)"
        ]

        for pattern in pet_patterns:
            matches = re.finditer(pattern, message, re.IGNORECASE)
            for match in matches:
                groups = match.groups()
                if len(groups) == 2:
                    if groups[0].lower() in ['dog', 'cat', 'bird', 'fish', 'hamster', 'rabbit', 'turtle', 'pet']:
                        pet_type = groups[0].lower()
                        pet_name = groups[1]
                    else:
                        pet_name = groups[0]
                        pet_type = groups[1].lower()

                    personal_info.append({
                        "key": f"pet_{pet_type}_name",
                        "value": pet_name,
                        "confidence": 0.9
                    })

        # Location extraction
        location_patterns = [
            r"(?:i|I) (?:live|reside|stay) in ([A-Za-z\s]+)",
            r"(?:i|I) am from ([A-Za-z\s]+)"
        ]

        for pattern in location_patterns:
            match = re.search(pattern, message, re.IGNORECASE)
            if match:
                location = match.group(1).strip()
                personal_info.append({
                    "key": "location",
                    "value": location,
                    "confidence": 0.8
                })

        return personal_info

    def store_memory(self, memory: UserMemoryCreate) -> UserMemory:
        """Store memory with conflict resolution"""
        # Check for existing similar memories
        existing = self.db.query(UserMemory).filter(
            and_(
                UserMemory.user_id == memory.user_id,
                UserMemory.key == memory.key
            )
        ).first()

        if existing:
            # Update if new memory has higher confidence
            if memory.confidence > existing.confidence:
                existing.value = memory.value
                existing.confidence = memory.confidence
                existing.updated_at = datetime.now()
                existing.source = memory.source
                self.db.commit()
                return existing
            else:
                return existing

        # Create new memory
        db_memory = UserMemory(**memory.dict())
        self.db.add(db_memory)
        self.db.commit()
        self.db.refresh(db_memory)

        logger.info(f"Stored memory: {memory.key} for user {memory.user_id}")
        return db_memory

    def store_memories(self, memories: List[UserMemoryCreate]) -> List[UserMemory]:
        """Store multiple memories efficiently"""
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
        """Get user memories with filtering"""
        query = self.db.query(UserMemory).filter(UserMemory.user_id == user_id)

        if memory_type:
            query = query.filter(UserMemory.memory_type == memory_type)

        if key_pattern:
            query = query.filter(UserMemory.key.like(f"%{key_pattern}%"))

        return query.order_by(desc(UserMemory.confidence), desc(UserMemory.last_accessed)).limit(limit).all()

    def get_memory_context(self, user_id: int) -> str:
        """Get memory context - deprecated, use enhanced version"""
        logger.warning("get_memory_context is deprecated, use get_consolidated_user_profile instead")
        
        memories = self.get_user_memories(user_id, limit=20)
        if not memories:
            return ""

        context_parts = []
        explicit_memories = [m for m in memories if m.memory_type == MemoryType.EXPLICIT and m.confidence >= 0.7]
        implicit_memories = [m for m in memories if m.memory_type == MemoryType.IMPLICIT and m.confidence >= 0.6]

        if explicit_memories:
            context_parts.append("Key facts:")
            for memory in explicit_memories[:5]:
                context_parts.append(f"- {memory.key}: {memory.value}")

        if implicit_memories:
            context_parts.append("Inferred preferences:")
            for memory in implicit_memories[:5]:
                context_parts.append(f"- {memory.key}: {memory.value}")

        return "\n".join(context_parts)

    def update_memory_access(self, memory_id: int):
        """Update memory access tracking"""
        memory = self.db.query(UserMemory).filter(UserMemory.id == memory_id).first()
        if memory:
            memory.last_accessed = datetime.now()
            memory.access_count += 1
            self.db.commit()

    def get_user_preferences(self, user_id: int) -> Dict[str, Any]:
        """Get user preferences"""
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
        """Set user preference"""
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
