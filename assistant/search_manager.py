"""
Fixed Cross-Conversation Search Manager for AI Assistant

This version addresses the key issues in the RAG pipeline:
1. Proper FTS table management and fallback search
2. Enhanced logging for debugging
3. Better keyword extraction and matching
4. Improved search result ranking and relevance
5. Structured historical context extraction
"""
import logging
import re
from typing import List, Optional, Dict, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_, desc, func, text
from datetime import datetime, timedelta
from llm_response_processor import LLMResponseProcessor
from lmstudio_client import lmstudio_client

logger = logging.getLogger(__name__)


class SearchManager:
    """Enhanced search manager with structured historical context"""

    def __init__(self, db: Session):
        self.db = db
        self.fts_available = self._check_fts_availability()
        self.response_processor = LLMResponseProcessor()
        logger.info(f"SearchManager initialized with FTS: {self.fts_available}")

    def _check_fts_availability(self) -> bool:
        """Check if FTS is available and functional"""
        try:
            # Check if FTS table exists
            result = self.db.execute(text(
                "SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_summaries_fts'"
            )).fetchone()

            if not result:
                logger.warning("FTS table 'conversation_summaries_fts' does not exist")
                return False

            # Test FTS functionality
            try:
                test_result = self.db.execute(text(
                    "SELECT COUNT(*) FROM conversation_summaries_fts LIMIT 1"
                )).fetchone()

                if test_result:
                    logger.info(f"FTS table is functional with {test_result[0]} entries")
                    return True
                else:
                    logger.warning("FTS table exists but is empty or non-functional")
                    return False

            except Exception as e:
                logger.error(f"FTS table exists but is not functional: {e}")
                return False

        except Exception as e:
            logger.error(f"Error checking FTS availability: {e}")
            return False

    def _rebuild_fts_if_needed(self):
        """Rebuild FTS table if it's empty but summaries exist"""
        try:
            from models import ConversationSummary

            # Check if FTS table is empty
            fts_count = self.db.execute(text("SELECT COUNT(*) FROM conversation_summaries_fts")).scalar()
            summary_count = self.db.query(ConversationSummary).count()

            logger.info(f"FTS entries: {fts_count}, Summary entries: {summary_count}")

            if fts_count == 0 and summary_count > 0:
                logger.info("Rebuilding FTS table...")

                # Clear FTS table
                self.db.execute(text("DELETE FROM conversation_summaries_fts"))

                # Insert all summaries
                summaries = self.db.query(ConversationSummary).all()
                for summary in summaries:
                    # Clean summary and keywords before storing in FTS
                    clean_summary = self.response_processor.process_summary_text(summary.summary or "")
                    clean_keywords = self.response_processor.process_summary_text(summary.keywords or "")

                    self.db.execute(text("""
                        INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                        VALUES (:id, :summary, :keywords)
                    """), {
                        "id": summary.id,
                        "summary": clean_summary,
                        "keywords": clean_keywords
                    })

                self.db.commit()
                logger.info(f"Rebuilt FTS table with {len(summaries)} entries")

        except Exception as e:
            logger.error(f"Failed to rebuild FTS table: {e}")

    async def search_conversations(
        self,
        user_id: int,
        query: str,
        limit: int = 5
    ) -> List:
        """Enhanced search with multiple strategies"""
        logger.info(f"Searching conversations for user_id {user_id} with query: '{query}'")

        try:
            from models import Conversation, ConversationSummary, Message

            # Rebuild FTS if needed
            if self.fts_available:
                self._rebuild_fts_if_needed()

            # Strategy 1: FTS search (if available)
            if self.fts_available:
                try:
                    fts_results = await self._search_with_fts(user_id, query, limit)
                    if fts_results:
                        logger.info(f"FTS search found {len(fts_results)} results")
                        return fts_results
                except Exception as e:
                    logger.warning(f"FTS search failed: {e}")

            # Strategy 2: Direct SQL search on summaries
            sql_results = await self._search_with_sql(user_id, query, limit)
            if sql_results:
                logger.info(f"SQL search found {len(sql_results)} results")
                return sql_results

            # Strategy 3: Conversation title search
            title_results = await self._search_by_title(user_id, query, limit)
            if title_results:
                logger.info(f"Title search found {len(title_results)} results")
                return title_results

            # Strategy 4: Message content search (fallback)
            content_results = await self._search_by_content(user_id, query, limit)
            if content_results:
                logger.info(f"Content search found {len(content_results)} results")
                return content_results

            logger.info("No search results found")
            return []

        except Exception as e:
            logger.error(f"Search failed: {e}")
            return []

    async def _search_with_fts(
        self,
        user_id: int,
        query: str,
        limit: int
    ) -> List:
        """Search using FTS with enhanced query processing"""
        try:
            from models import ConversationSummary

            # Prepare FTS query
            fts_query = self._prepare_fts_query(query)
            logger.info(f"FTS query: '{fts_query}'")

            # Execute FTS search
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

            logger.info(f"FTS search returned {len(summaries)} summaries")
            return summaries

        except Exception as e:
            logger.error(f"FTS search failed: {e}")
            raise

    async def _search_with_sql(
        self,
        user_id: int,
        query: str,
        limit: int
    ) -> List:
        """Search using SQL LIKE with enhanced term matching"""
        try:
            from models import Conversation, ConversationSummary

            # Extract search terms
            terms = self._extract_search_terms(query)
            logger.info(f"Search terms: {terms}")

            if not terms:
                return []

            # Build conditions for each term
            conditions = []
            for term in terms:
                term_conditions = [
                    ConversationSummary.summary.ilike(f"%{term}%"),
                    ConversationSummary.keywords.ilike(f"%{term}%"),
                    Conversation.title.ilike(f"%{term}%")
                ]
                conditions.extend(term_conditions)

            # Search with OR conditions
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

            logger.info(f"SQL search returned {len(summaries)} summaries")
            return summaries

        except Exception as e:
            logger.error(f"SQL search failed: {e}")
            return []

    async def _search_by_title(
        self,
        user_id: int,
        query: str,
        limit: int
    ) -> List:
        """Search by conversation title with pseudo-summary creation"""
        try:
            from models import Conversation, ConversationSummary

            terms = self._extract_search_terms(query)
            if not terms:
                return []

            # Build title conditions
            title_conditions = []
            for term in terms:
                title_conditions.append(Conversation.title.ilike(f"%{term}%"))

            # Find matching conversations
            conversations = self.db.query(Conversation).filter(
                and_(
                    Conversation.user_id == user_id,
                    Conversation.is_active == True,
                    or_(*title_conditions)
                )
            ).order_by(desc(Conversation.updated_at)).limit(limit).all()

            # Convert to summaries (create pseudo-summaries if needed)
            summaries = []
            for conv in conversations:
                existing_summary = self.db.query(ConversationSummary).filter(
                    ConversationSummary.conversation_id == conv.id
                ).first()

                if existing_summary:
                    summaries.append(existing_summary)
                else:
                    # Create pseudo-summary for search results
                    from models import ConversationSummary as SummaryModel
                    pseudo_summary = SummaryModel(
                        conversation_id=conv.id,
                        summary=f"Conversation about {conv.title}",
                        keywords=conv.title.lower(),
                        message_count=0,
                        priority_score=0.5
                    )
                    summaries.append(pseudo_summary)

            logger.info(f"Title search returned {len(summaries)} summaries")
            return summaries

        except Exception as e:
            logger.error(f"Title search failed: {e}")
            return []

    async def _search_by_content(
        self,
        user_id: int,
        query: str,
        limit: int
    ) -> List:
        """Search by message content as final fallback"""
        try:
            from models import Conversation, ConversationSummary, Message

            terms = self._extract_search_terms(query)
            if not terms:
                return []

            # Build content conditions
            content_conditions = []
            for term in terms:
                content_conditions.append(Message.content.ilike(f"%{term}%"))

            # Find conversations with matching messages
            matching_conversations = self.db.query(Conversation).join(Message).filter(
                and_(
                    Conversation.user_id == user_id,
                    Conversation.is_active == True,
                    or_(*content_conditions)
                )
            ).distinct().order_by(desc(Conversation.updated_at)).limit(limit).all()

            # Convert to summaries
            summaries = []
            for conv in matching_conversations:
                existing_summary = self.db.query(ConversationSummary).filter(
                    ConversationSummary.conversation_id == conv.id
                ).first()

                if existing_summary:
                    summaries.append(existing_summary)
                else:
                    # Create pseudo-summary
                    from models import ConversationSummary as SummaryModel
                    pseudo_summary = SummaryModel(
                        conversation_id=conv.id,
                        summary=f"Conversation containing relevant content",
                        keywords=", ".join(terms),
                        message_count=0,
                        priority_score=0.3
                    )
                    summaries.append(pseudo_summary)

            logger.info(f"Content search returned {len(summaries)} summaries")
            return summaries

        except Exception as e:
            logger.error(f"Content search failed: {e}")
            return []

    def _prepare_fts_query(self, query: str) -> str:
        """Prepare query for FTS search"""
        # Remove special characters and normalize
        clean_query = re.sub(r'[^\w\s]', ' ', query)
        terms = clean_query.lower().split()

        # Remove very short terms
        terms = [term for term in terms if len(term) >= 2]

        # Join with OR for broader matching
        if len(terms) > 1:
            return " OR ".join(terms)
        elif len(terms) == 1:
            return terms[0]
        else:
            return "python"  # Fallback term

    def _extract_search_terms(self, query: str) -> List[str]:
        """Extract meaningful search terms from query"""
        # Remove punctuation and split
        words = re.findall(r'\b[a-zA-Z]{2,}\b', query.lower())

        # Remove common stop words
        stop_words = {
            'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by',
            'this', 'that', 'these', 'those', 'a', 'an', 'are', 'is', 'was', 'were',
            'have', 'has', 'had', 'will', 'would', 'could', 'should', 'can', 'may', 'might',
            'must', 'shall', 'do', 'does', 'did', 'get', 'got', 'go', 'goes', 'went'
        }

        meaningful_terms = [word for word in words if word not in stop_words and len(word) >= 2]

        # Return top 10 terms to avoid overly complex queries
        return meaningful_terms[:10]

    async def get_structured_historical_context(
        self,
        user_id: int,
        current_message: str,
        limit: int = 2
    ) -> Dict[str, Any]:
        """Get structured historical context with actionable summaries"""
        logger.info(f"Getting structured historical context for user {user_id}")

        try:
            # Extract keywords from current message
            keywords = await self.extract_keywords(current_message)
            logger.info(f"Extracted keywords: {keywords}")

            if not keywords:
                logger.info("No keywords extracted, returning empty context")
                return {}

            # Search for related conversations
            search_query = " ".join(keywords[:5])
            related_conversations = await self.search_conversations(user_id, search_query, limit)

            if not related_conversations:
                return {}

            # Structure the historical context
            structured_context = {
                "similar_topics": [],
                "relevant_solutions": [],
                "user_patterns": [],
                "project_continuations": []
            }

            # Process each conversation to extract structured insights
            for conv_summary in related_conversations:
                insights = await self._extract_structured_insights(conv_summary)
                self._categorize_insights(insights, structured_context)

            # Remove empty categories
            structured_context = {k: v for k, v in structured_context.items() if v}

            logger.info(f"Generated structured context with {len(structured_context)} categories")
            return structured_context

        except Exception as e:
            logger.error(f"Failed to get structured historical context: {e}")
            return {}

    async def get_related_conversations(
        self,
        user_id: int,
        message: str,
        limit: int = 3
    ) -> List:
        """Get conversations related to current message - legacy method"""
        logger.info(f"Getting related conversations for user {user_id}")

        try:
            # Extract keywords from message
            keywords = await self.extract_keywords(message)
            logger.info(f"Extracted keywords: {keywords}")

            if not keywords:
                logger.info("No keywords extracted, returning empty results")
                return []

            # Create search query from keywords
            search_query = " ".join(keywords[:5])  # Use top 5 keywords
            logger.info(f"Search query: '{search_query}'")

            # Search for related conversations
            related = await self.search_conversations(user_id, search_query, limit)
            logger.info(f"Found {len(related)} related conversations")

            return related

        except Exception as e:
            logger.error(f"Failed to get related conversations: {e}")
            return []

    async def extract_keywords(self, text: str) -> List[str]:
        """Enhanced keyword extraction"""
        try:
            # Use enhanced extraction
            keywords = self._extract_keywords_enhanced(text)
            logger.info(f"Extracted {len(keywords)} keywords: {keywords}")
            return keywords

        except Exception as e:
            logger.error(f"Keyword extraction failed: {e}")
            return []

    def _extract_keywords_enhanced(self, text: str) -> List[str]:
        """Enhanced keyword extraction with better relevance scoring"""
        # Extract all words
        words = re.findall(r'\b[a-zA-Z]{2,}\b', text.lower())

        # Stop words to filter out
        stop_words = {
            'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by',
            'this', 'that', 'these', 'those', 'a', 'an', 'are', 'is', 'was', 'were',
            'have', 'has', 'had', 'will', 'would', 'could', 'should', 'can', 'may', 'might',
            'must', 'shall', 'do', 'does', 'did', 'get', 'got', 'go', 'goes', 'went',
            'also', 'just', 'now', 'then', 'than', 'only', 'very', 'well', 'still',
            'about', 'into', 'through', 'during', 'before', 'after', 'above', 'below',
            'up', 'down', 'out', 'off', 'over', 'under', 'again', 'further', 'then', 'once'
        }

        # Technical terms that should be prioritized
        technical_terms = {
            'python', 'javascript', 'java', 'code', 'programming', 'function', 'method',
            'class', 'object', 'variable', 'array', 'list', 'dictionary', 'string',
            'exception', 'error', 'debug', 'test', 'api', 'database', 'server',
            'client', 'framework', 'library', 'algorithm', 'data', 'structure',
            'web', 'app', 'application', 'software', 'development', 'frontend',
            'backend', 'deployment', 'docker', 'kubernetes', 'aws', 'cloud'
        }

        # Filter and score words
        word_scores = {}
        for word in words:
            if word not in stop_words and len(word) >= 2:
                score = 1.0

                # Boost technical terms
                if word in technical_terms:
                    score += 2.0

                # Boost longer words
                if len(word) >= 6:
                    score += 0.5

                # Boost capitalized words (proper nouns)
                if word[0].isupper():
                    score += 0.3

                word_scores[word] = word_scores.get(word, 0) + score

        # Sort by score and return top keywords
        sorted_keywords = sorted(word_scores.items(), key=lambda x: x[1], reverse=True)

        # Return top keywords
        return [word for word, score in sorted_keywords[:10]]

    async def _extract_structured_insights(self, conv_summary) -> Dict[str, Any]:
        """Extract structured insights from a conversation summary"""
        try:
            # Use LLM to extract structured insights
            insights_prompt = f"""
            Analyze this conversation summary and extract structured insights:

            Summary: {conv_summary.summary}
            Keywords: {conv_summary.keywords}

            Extract insights in these categories:
            1. similar_topics: Topics or themes discussed
            2. relevant_solutions: Solutions, answers, or helpful information provided
            3. user_patterns: User behavior patterns or preferences shown
            4. project_continuations: Ongoing projects or goals mentioned

            Return a JSON object with these categories as keys and arrays of insights as values.
            Keep insights concise (max 80 chars each).

            Example:
            {{
                "similar_topics": ["Python programming", "file handling"],
                "relevant_solutions": ["Use os.path.join() for file paths"],
                "user_patterns": ["Prefers detailed explanations"],
                "project_continuations": ["Working on data processing script"]
            }}
            """

            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are an insight extraction system. Return only valid JSON."},
                    {"role": "user", "content": insights_prompt}
                ],
                temperature=0.1,
                max_tokens=1000
            )

            # Process response to remove thinking tags
            processed_response = self.response_processor.process_chat_response(response)
            content = processed_response["choices"][0]["message"]["content"].strip()

            # Extract and validate JSON
            import json
            try:
                insights = json.loads(content)
                # Validate structure
                if isinstance(insights, dict):
                    return insights
                else:
                    logger.warning(f"Invalid insights structure: {insights}")
                    return {}
            except json.JSONDecodeError:
                logger.warning(f"Failed to parse insights JSON: {content}")
                return {}

        except Exception as e:
            logger.error(f"Failed to extract structured insights: {e}")
            return {}

    def _categorize_insights(self, insights: Dict[str, Any], structured_context: Dict[str, Any]):
        """Categorize insights into the structured context"""
        for category, items in insights.items():
            if category in structured_context and isinstance(items, list):
                # Add items to the appropriate category, avoiding duplicates
                for item in items:
                    if isinstance(item, str) and item not in structured_context[category]:
                        structured_context[category].append(item)

    def calculate_conversation_priority(self, conversation_id: int) -> float:
        """Calculate priority score for a conversation"""
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

            # Calculate priority based on multiple factors
            priority = 0.0

            # Factor 1: Message count (more messages = higher priority)
            message_count = len(messages)
            priority += min(message_count * 0.1, 0.5)

            # Factor 2: Recency (more recent = higher priority)
            days_ago = (datetime.now() - conversation.updated_at).days
            recency_score = max(0, 1 - (days_ago / 30))  # Decay over 30 days
            priority += recency_score * 0.3

            # Factor 3: User engagement (longer messages = higher priority)
            total_length = sum(len(msg.content) for msg in messages)
            avg_length = total_length / message_count if message_count > 0 else 0
            engagement_score = min(avg_length / 1000, 1.0)  # Normalize to 1000 chars
            priority += engagement_score * 0.2

            # Ensure priority is between 0 and 1
            return min(priority, 1.0)

        except Exception as e:
            logger.error(f"Failed to calculate conversation priority: {e}")
            return 0.0

    def rebuild_search_index(self, user_id: Optional[int] = None):
        """Rebuild search index for better performance"""
        try:
            logger.info("Rebuilding search index...")

            if not self.fts_available:
                logger.warning("FTS not available, cannot rebuild index")
                return

            from models import ConversationSummary, Conversation

            # Get summaries to rebuild
            query = self.db.query(ConversationSummary).join(Conversation)

            if user_id:
                query = query.filter(Conversation.user_id == user_id)

            summaries = query.all()

            # Clear FTS table
            self.db.execute(text("DELETE FROM conversation_summaries_fts"))

            # Rebuild FTS entries
            for summary in summaries:
                # Clean summary and keywords before storing in FTS
                clean_summary = self.response_processor.process_summary_text(summary.summary or "")
                clean_keywords = self.response_processor.process_summary_text(summary.keywords or "")

                self.db.execute(text("""
                    INSERT INTO conversation_summaries_fts(rowid, summary, keywords)
                    VALUES (:id, :summary, :keywords)
                """), {
                    "id": summary.id,
                    "summary": clean_summary,
                    "keywords": clean_keywords
                })

            self.db.commit()
            logger.info(f"Rebuilt search index with {len(summaries)} entries")

        except Exception as e:
            logger.error(f"Failed to rebuild search index: {e}")
