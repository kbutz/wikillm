"""
Cross-Conversation Search Manager for AI Assistant - Fixed Version
"""
import logging
import re
from typing import List, Optional, Dict, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc, func, text
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class SearchManager:
    """Manages cross-conversation search and context retrieval"""
    
    def __init__(self, db: Session):
        self.db = db
        self.fts_available = self._ensure_fts_table()
    
    def _ensure_fts_table(self):
        """Ensure FTS virtual table exists for full-text search"""
        try:
            # Check if FTS table exists
            result = self.db.execute(text("SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_summaries_fts'")).fetchone()
            if not result:
                logger.warning("FTS table not found, will use fallback search")
                return False
            
            # Test FTS table functionality
            test_result = self.db.execute(text("SELECT COUNT(*) FROM conversation_summaries_fts LIMIT 1")).fetchone()
            if test_result:
                logger.info("FTS table is functional")
                return True
            else:
                logger.warning("FTS table exists but is not functional")
                return False
                
        except Exception as e:
            logger.warning(f"Could not check FTS table: {e}")
            return False
    
    async def _search_conversations_fts(
        self,
        user_id: int,
        query: str,
        limit: int = 5
    ) -> List:
        """Search conversations using FTS"""
        try:
            from models import Conversation, ConversationSummary
            
            # Prepare FTS query (escape special characters)
            fts_query = query.replace('"', '""').replace("'", "''")
            
            # Use FTS to find matching summaries
            fts_results = self.db.execute(text("""
                SELECT cs.id, cs.conversation_id, cs.summary, cs.keywords, cs.priority_score,
                       bm25(conversation_summaries_fts) as rank
                FROM conversation_summaries_fts 
                JOIN conversation_summaries cs ON cs.id = conversation_summaries_fts.rowid
                JOIN conversations c ON c.id = cs.conversation_id
                WHERE conversation_summaries_fts MATCH :query
                  AND c.user_id = :user_id
                  AND c.is_active = 1
                ORDER BY rank, cs.priority_score DESC
                LIMIT :limit
            """), {
                "query": fts_query,
                "user_id": user_id,
                "limit": limit
            }).fetchall()
            
            # Convert to ConversationSummary objects
            summaries = []
            for row in fts_results:
                summary = self.db.query(ConversationSummary).filter(
                    ConversationSummary.id == row.id
                ).first()
                if summary:
                    summaries.append(summary)
            
            return summaries
            
        except Exception as e:
            logger.error(f"FTS search failed: {e}")
            raise
    
    async def search_conversations(
        self,
        user_id: int,
        query: str,
        limit: int = 5
    ) -> List:
        """Search conversations using FTS when available, fallback to LIKE search"""
        try:
            from models import Conversation, ConversationSummary
            
            # Try FTS search first
            try:
                fts_results = await self._search_conversations_fts(user_id, query, limit)
                if fts_results:
                    logger.info(f"Found {len(fts_results)} conversations using FTS for query: {query}")
                    return fts_results
            except Exception as e:
                logger.warning(f"FTS search failed, falling back to LIKE search: {e}")
            
            # Fallback to LIKE search
            query_terms = query.lower().split()
            
            # Build OR conditions for each term
            conditions = []
            for term in query_terms[:5]:  # Limit to first 5 terms
                if len(term) > 2:  # Skip very short terms
                    conditions.extend([
                        ConversationSummary.summary.ilike(f"%{term}%"),
                        ConversationSummary.keywords.ilike(f"%{term}%"),
                        Conversation.title.ilike(f"%{term}%")
                    ])
            
            if not conditions:
                return []
            
            summaries = self.db.query(ConversationSummary).join(Conversation).filter(
                and_(
                    Conversation.user_id == user_id,
                    Conversation.is_active == True,
                    or_(*conditions)
                )
            ).order_by(
                desc(ConversationSummary.priority_score),
                desc(Conversation.updated_at)
            ).limit(limit).all()
            
            logger.info(f"Found {len(summaries)} conversations using LIKE search for query: {query}")
            return summaries
            
        except Exception as e:
            logger.error(f"Search failed: {e}")
            return []
    
    async def get_related_conversations(
        self,
        user_id: int,
        message: str,
        limit: int = 3
    ) -> List:
        """Get conversations related to current message context"""
        try:
            # Extract key terms from message
            keywords = await self.extract_keywords(message)
            
            if not keywords:
                return []
            
            # Search using extracted keywords
            keyword_query = ' '.join(keywords[:5])  # Use top 5 keywords
            return await self.search_conversations(user_id, keyword_query, limit)
        except Exception as e:
            logger.error(f"Failed to get related conversations: {e}")
            return []
    
    async def extract_keywords(self, text: str) -> List[str]:
        """Extract keywords from text using simple method"""
        try:
            # Simple keyword extraction fallback
            return self._extract_keywords_simple(text)
        except Exception as e:
            logger.error(f"Keyword extraction failed: {e}")
            return []
    
    def _extract_keywords_simple(self, text: str) -> List[str]:
        """Simple keyword extraction fallback"""
        # Remove punctuation and split
        words = re.findall(r'\\b[a-zA-Z]{3,}\\b', text.lower())
        
        # Remove common stop words
        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by', 'this', 'that', 'are', 'is', 'was', 'were', 'have', 'has', 'had', 'will', 'would', 'could', 'should'}
        keywords = [word for word in words if word not in stop_words]
        
        # Return unique keywords, prioritizing longer words
        unique_keywords = []
        for keyword in sorted(set(keywords), key=len, reverse=True):
            if keyword not in unique_keywords:
                unique_keywords.append(keyword)
        
        return unique_keywords[:7]
    
    async def get_user_priorities(self, user_id: int) -> Dict[str, Any]:
        """Extract user priorities from conversation history"""
        try:
            from models import Conversation, ConversationSummary
            
            # Get recent conversations
            priority_conversations = self.db.query(ConversationSummary).join(Conversation).filter(
                and_(
                    Conversation.user_id == user_id,
                    Conversation.is_active == True
                )
            ).order_by(desc(Conversation.updated_at)).limit(5).all()
            
            if not priority_conversations:
                return {
                    "priorities": [],
                    "goals": [],
                    "interests": []
                }
            
            # Simple priority extraction from titles and summaries
            all_text = " ".join([
                f"{conv.conversation.title} {conv.summary}" 
                for conv in priority_conversations
            ]).lower()
            
            priorities = self._extract_simple_priorities(all_text)
            
            return {
                "priorities": priorities[:5],
                "goals": [],
                "interests": []
            }
        
        except Exception as e:
            logger.error(f"Priority extraction failed: {e}")
            return {
                "priorities": ["No priorities found"],
                "goals": [],
                "interests": []
            }
    
    def _extract_simple_priorities(self, text: str) -> List[str]:
        """Simple priority extraction from text"""
        # Look for action words and important nouns
        priority_patterns = [
            r'(?:need to|want to|planning to|working on|focusing on) ([^.!?]+)',
            r'(?:priority|important|urgent|goal) (?:is|to) ([^.!?]+)',
            r'(?:learning|studying|building|creating|developing) ([^.!?]+)'
        ]
        
        priorities = []
        for pattern in priority_patterns:
            matches = re.findall(pattern, text.lower())
            priorities.extend([match.strip() for match in matches if len(match.strip()) > 5])
        
        return list(set(priorities))[:5]  # Return unique priorities, max 5
    
    def calculate_conversation_priority(self, conversation_id: int) -> float:
        """Calculate priority score based on conversation characteristics"""
        try:
            from models import Conversation, Message
            
            conversation = self.db.query(Conversation).filter(
                Conversation.id == conversation_id
            ).first()
            
            if not conversation:
                return 0.0
                
            messages = self.db.query(Message).filter(
                Message.conversation_id == conversation_id
            ).all()
            
            if not messages:
                return 0.0
            
            priority_score = 0.0
            
            # Recent activity
            if conversation.updated_at > datetime.now() - timedelta(days=7):
                priority_score += 0.3
            
            # Message count
            message_count = len(messages)
            if message_count > 10:
                priority_score += 0.2
            elif message_count > 5:
                priority_score += 0.1
            
            # Important keywords
            important_keywords = [
                'urgent', 'important', 'priority', 'deadline', 'asap', 'critical',
                'project', 'work', 'goal', 'plan', 'need help', 'problem', 'issue'
            ]
            
            full_conversation_text = ' '.join([msg.content.lower() for msg in messages])
            keyword_matches = sum(1 for keyword in important_keywords if keyword in full_conversation_text)
            
            if keyword_matches > 0:
                priority_score += min(0.3, keyword_matches * 0.1)
            
            return min(1.0, priority_score)
        
        except Exception as e:
            logger.error(f"Priority calculation failed: {e}")
            return 0.0
    
    def update_conversation_priority(self, conversation_id: int, priority_score: float):
        """Update priority score for a conversation"""
        try:
            from models import ConversationSummary
            
            summary = self.db.query(ConversationSummary).filter(
                ConversationSummary.conversation_id == conversation_id
            ).first()
            
            if summary:
                summary.priority_score = max(0.0, min(1.0, priority_score))
                summary.updated_at = datetime.now()
                self.db.commit()
                logger.info(f"Updated priority score for conversation {conversation_id}: {priority_score}")
        except Exception as e:
            logger.error(f"Failed to update priority score: {e}")
    
    def rebuild_search_index(self, user_id: Optional[int] = None):
        """Rebuild the FTS search index"""
        try:
            logger.info("Search index rebuild requested but not implemented in fallback mode")
        except Exception as e:
            logger.error(f"Failed to rebuild search index: {e}")
