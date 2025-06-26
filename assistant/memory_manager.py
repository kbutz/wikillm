"""
Memory Management System for User Personalization
"""
import logging
from typing import Dict, List, Optional, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc, func
from datetime import datetime, timedelta
from models import UserMemory, UserPreference, User
from schemas import UserMemoryCreate, UserPreferenceCreate, MemoryType
import json
import re

logger = logging.getLogger(__name__)


class MemoryManager:
    """Manages user memory and personalization"""
    
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
        
        # Task breakdown preferences
        if any(phrase in message.lower() for phrase in ["step by step", "break down", "one at a time"]):
            preferences.append({
                "key": "task_breakdown",
                "value": "step_by_step",
                "confidence": 0.8
            })
        
        return preferences
    
    def _extract_personal_info(self, message: str) -> List[Dict[str, Any]]:
        """Extract personal information from user messages"""
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
        
        # Job/profession extraction
        job_patterns = [
            r"(?:i work as|i'm a|i am a|my job is|i do) ([a-zA-Z\s]+)",
            r"(?:i'm|i am) (?:a|an) ([a-zA-Z\s]+) (?:by profession|for work)"
        ]
        
        for pattern in job_patterns:
            match = re.search(pattern, message, re.IGNORECASE)
            if match:
                job = match.group(1).strip()
                if len(job.split()) <= 3:  # Reasonable job title length
                    personal_info.append({
                        "key": "profession",
                        "value": job,
                        "confidence": 0.8
                    })
        
        # Location extraction
        location_patterns = [
            r"(?:i live in|i'm from|based in|located in) ([A-Za-z\s,]+)",
            r"(?:from|in) ([A-Z][a-z]+(?:,\s*[A-Z][a-z]+)?)"
        ]
        
        for pattern in location_patterns:
            match = re.search(pattern, message, re.IGNORECASE)
            if match:
                location = match.group(1).strip()
                if 2 <= len(location) <= 50:  # Reasonable location length
                    personal_info.append({
                        "key": "location",
                        "value": location,
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
            context_parts.append("\\nBased on conversation patterns:")
            for memory in implicit_memories[:10]:  # Top 10 implicit
                if memory.confidence > 0.5:  # Only include confident predictions
                    context_parts.append(f"- {memory.key}: {memory.value}")
        
        return "\\n".join(context_parts)
    
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
    
    def cleanup_old_memories(self, user_id: int, days_old: int = 90):
        """Clean up old, low-confidence memories"""
        cutoff_date = datetime.now() - timedelta(days=days_old)
        
        old_memories = self.db.query(UserMemory).filter(
            and_(
                UserMemory.user_id == user_id,
                UserMemory.confidence < 0.3,
                UserMemory.last_accessed < cutoff_date
            )
        ).all()
        
        for memory in old_memories:
            self.db.delete(memory)
        
        if old_memories:
            logger.info(f"Cleaned up {len(old_memories)} old memories for user {user_id}")
            self.db.commit()
