"""
Conversation Management Service - Enhanced Version with Consolidated User Profiles
"""
import logging
from typing import List, Optional, Dict, Any, Tuple
from sqlalchemy.orm import Session
from sqlalchemy import desc, and_
from datetime import datetime, timedelta
from collections import Counter
from models import Conversation, Message, User, ConversationSummary
from schemas import ConversationCreate, MessageCreate, MessageRole
from memory_manager import MemoryManager
from lmstudio_client import lmstudio_client
from llm_response_processor import LLMResponseProcessor
import asyncio

logger = logging.getLogger(__name__)


class ConversationManager:
    """Manages conversations and message history with enhanced context building"""

    def __init__(self, db: Session):
        self.db = db
        self.memory_manager = MemoryManager(db)
        self.response_processor = LLMResponseProcessor()

    def create_conversation(self, user_id: int, title: str = None) -> Conversation:
        """Create a new conversation"""
        if not title:
            # Generate a title based on timestamp
            title = f"Conversation {datetime.now().strftime('%Y-%m-%d %H:%M')}"

        conversation = Conversation(
            user_id=user_id,
            title=title
        )
        self.db.add(conversation)
        self.db.commit()
        self.db.refresh(conversation)

        logger.info(f"Created new conversation {conversation.id} for user {user_id}")
        return conversation

    def get_conversation(self, conversation_id: int, user_id: int) -> Optional[Conversation]:
        """Get a specific conversation with access control"""
        return self.db.query(Conversation).filter(
            and_(
                Conversation.id == conversation_id,
                Conversation.user_id == user_id,
                Conversation.is_active == True
            )
        ).first()

    def get_user_conversations(self, user_id: int, limit: int = 50) -> List[Conversation]:
        """Get all conversations for a user"""
        return self.db.query(Conversation).filter(
            and_(
                Conversation.user_id == user_id,
                Conversation.is_active == True
            )
        ).order_by(desc(Conversation.updated_at)).limit(limit).all()

    def add_message(
        self,
        conversation_id: int,
        role: MessageRole,
        content: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Message:
        """Add a message to a conversation"""
        message = Message(
            conversation_id=conversation_id,
            role=role,
            content=content
        )

        # Add metadata if provided
        if metadata:
            message.token_count = metadata.get("token_count")
            message.llm_model = metadata.get("model_used")
            message.temperature = metadata.get("temperature")
            message.processing_time = metadata.get("processing_time")

        self.db.add(message)

        # Update conversation timestamp
        conversation = self.db.query(Conversation).filter(
            Conversation.id == conversation_id
        ).first()
        if conversation:
            conversation.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(message)

        return message

    def get_conversation_messages(
        self,
        conversation_id: int,
        limit: int = None,
        offset: int = 0
    ) -> List[Message]:
        """Get messages from a conversation"""
        query = self.db.query(Message).filter(
            Message.conversation_id == conversation_id
        ).order_by(Message.timestamp)

        if limit:
            query = query.offset(offset).limit(limit)

        return query.all()

    def get_recent_messages(
        self,
        conversation_id: int,
        max_messages: int = 20
    ) -> List[Message]:
        """Get recent messages for context"""
        return self.db.query(Message).filter(
            Message.conversation_id == conversation_id
        ).order_by(desc(Message.timestamp)).limit(max_messages).all()[::-1]  # Reverse to chronological order

    async def build_conversation_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 20,
        include_historical_context: bool = True
    ) -> List[Dict[str, str]]:
        """Build conversation context for LLM with enhanced user profiles and structured history"""
        # Use enhanced memory manager if available
        try:
            from memory_manager import EnhancedMemoryManager
            enhanced_memory = EnhancedMemoryManager(self.db)
            use_enhanced = True
        except ImportError:
            enhanced_memory = None
            use_enhanced = False

        # Get user memory context from consolidated profile
        if use_enhanced:
            try:
                user_profile = await enhanced_memory.get_consolidated_user_profile(user_id)
                memory_context = self._format_user_profile_for_context(user_profile)
            except Exception as e:
                logger.warning(f"Could not get consolidated user profile: {e}")
                memory_context = self.memory_manager.get_memory_context(user_id)
        else:
            memory_context = self.memory_manager.get_memory_context(user_id)

        # Get recent messages
        messages = self.get_recent_messages(conversation_id, max_messages)

        # Get contextual memories based on current conversation
        contextual_memory = ""
        if use_enhanced and enhanced_memory and messages:
            # Get the most recent user message for context
            current_message = ""
            for msg in reversed(messages):
                if msg.role == MessageRole.USER:
                    current_message = msg.content
                    break

            if current_message:
                try:
                    contextual_memory = await enhanced_memory.get_contextual_memories(
                        user_id, current_message, limit=3
                    )
                except Exception as e:
                    logger.warning(f"Could not get contextual memories: {e}")

        # Get structured historical context if enabled
        historical_context = ""
        if include_historical_context and messages:
            try:
                from search_manager import SearchManager
                search_manager = SearchManager(self.db)

                # Get the most recent user message for context
                current_message = ""
                for msg in reversed(messages):
                    if msg.role == MessageRole.USER:
                        current_message = msg.content
                        break

                if current_message:
                    structured_context = await search_manager.get_structured_historical_context(
                        user_id, current_message, limit=2
                    )

                    if structured_context:
                        historical_parts = []

                        # Format relevant solutions
                        if structured_context.get("relevant_solutions"):
                            historical_parts.append("Previous solutions:")
                            for solution in structured_context["relevant_solutions"][:2]:
                                historical_parts.append(f"• {solution}")

                        # Format project continuations
                        if structured_context.get("project_continuations"):
                            historical_parts.append("Project continuations:")
                            for continuation in structured_context["project_continuations"][:2]:
                                historical_parts.append(f"• {continuation}")

                        # Format similar topics
                        if structured_context.get("similar_topics"):
                            topics = ", ".join(structured_context["similar_topics"][:4])
                            historical_parts.append(f"Related topics: {topics}")

                        if historical_parts:
                            historical_context = f"\\n\\nFrom previous conversations:\\n{chr(10).join(historical_parts)}"

            except Exception as e:
                logger.warning(f"Could not get structured historical context: {e}")
                historical_context = ""

        # Build context array
        context = []

        # Add system context with user memory and historical context
        system_parts = ["You are a helpful AI assistant."]

        if memory_context:
            system_parts.append(f"Here's what you know about the user:\\n{memory_context}")

        if contextual_memory:
            system_parts.append(f"\\n{contextual_memory}")

        if historical_context:
            system_parts.append(historical_context)

        if len(system_parts) > 1:
            system_parts.append("\\nUse this information to provide personalized and contextually relevant responses.")

        context.append({"role": "system", "content": "\\n\\n".join(system_parts)})

        # Add conversation messages
        for message in messages:
            context.append({
                "role": message.role,
                "content": message.content
            })

        logger.info(f"Built context for conversation {conversation_id} with {len(context)} messages for user {user_id}")

        return context

    def _format_user_profile_for_context(self, user_profile: Dict[str, Any]) -> str:
        """Format consolidated user profile for conversation context"""
        if not user_profile:
            return ""

        context_parts = []

        # Personal information
        if user_profile.get("personal"):
            personal_items = [f"{k}: {v}" for k, v in user_profile["personal"].items()]
            if personal_items:
                context_parts.append("Personal: " + "; ".join(personal_items[:3]))

        # Preferences
        if user_profile.get("preferences"):
            pref_items = [f"{k}: {v}" for k, v in user_profile["preferences"].items()]
            if pref_items:
                context_parts.append("Preferences: " + "; ".join(pref_items[:3]))

        # Skills
        if user_profile.get("skills"):
            skill_items = [f"{k}: {v}" for k, v in user_profile["skills"].items()]
            if skill_items:
                context_parts.append("Skills: " + "; ".join(skill_items[:3]))

        # Current projects
        if user_profile.get("projects"):
            project_items = [f"{k}: {v}" for k, v in user_profile["projects"].items()]
            if project_items:
                context_parts.append("Projects: " + "; ".join(project_items[:2]))

        # Context
        if user_profile.get("context"):
            context_items = [f"{k}: {v}" for k, v in user_profile["context"].items()]
            if context_items:
                context_parts.append("Context: " + "; ".join(context_items[:2]))

        return "\\n".join(context_parts) if context_parts else ""

    def delete_conversation(self, conversation_id: int, user_id: int) -> bool:
        """Soft delete a conversation"""
        conversation = self.get_conversation(conversation_id, user_id)
        if not conversation:
            return False

        conversation.is_active = False
        self.db.commit()

        logger.info(f"Deleted conversation {conversation_id} for user {user_id}")
        return True

    def update_conversation_title(self, conversation_id: int, user_id: int, title: str) -> bool:
        """Update conversation title"""
        conversation = self.get_conversation(conversation_id, user_id)
        if not conversation:
            return False

        conversation.title = title
        conversation.updated_at = datetime.now()
        self.db.commit()

        return True

    async def generate_conversation_title(self, conversation_id: int) -> Optional[str]:
        """Generate a smart title based on conversation content"""
        messages = self.get_conversation_messages(conversation_id, limit=5)

        if not messages:
            logger.warning(f"No messages found for conversation {conversation_id} when generating title")
            return None

        # Get first few user messages for context
        user_messages = [msg.content for msg in messages if msg.role == MessageRole.USER][:3]

        if not user_messages:
            logger.warning(f"No user messages found for conversation {conversation_id} when generating title")
            return None

        # Create a prompt to generate title
        context = [
            {
                "role": "system",
                "content": "Generate a short, descriptive title (max 50 characters) for this conversation based on the user's messages. Return only the title, no quotes or extra text."
            },
            {
                "role": "user",
                "content": f"Generate a title for a conversation that starts with these messages: {' | '.join(user_messages)}"
            }
        ]

        try:
            # Log the request for diagnostics
            logger.info(f"Generating title for conversation {conversation_id} with {len(user_messages)} user messages")

            response = await lmstudio_client.chat_completion(
                messages=context,
                temperature=0.3,
                max_tokens=50
            )

            # Process the response to remove thinking tags
            processed_response = self.response_processor.process_chat_response(response)

            # Extract title from processed response
            raw_title = processed_response["choices"][0]["message"]["content"].strip()

            # Log the raw title for diagnostics
            logger.debug(f"Raw title generated for conversation {conversation_id}: '{raw_title}'")

            # Check if the title still contains thinking tags
            if "<think>" in raw_title or "</think>" in raw_title:
                logger.warning(f"Title still contains thinking tags after processing for conversation {conversation_id}")
                # Apply additional thinking tag removal
                raw_title = self.response_processor.remove_thinking_tags(raw_title)
                logger.debug(f"Title after additional thinking tag removal: '{raw_title}'")

            # Force remove any remaining thinking tags with a direct regex replacement
            import re
            raw_title = re.sub(r'<think>.*?</think>', '', raw_title, flags=re.DOTALL | re.IGNORECASE)
            raw_title = re.sub(r'<think>.*', '', raw_title, flags=re.DOTALL | re.IGNORECASE)  # Unclosed tags
            raw_title = re.sub(r'.*</think>', '', raw_title, flags=re.DOTALL | re.IGNORECASE)  # Unopened tags

            # Check if the title is empty or too short after processing
            if not raw_title or len(raw_title) < 3:
                logger.warning(f"Empty or too short title generated for conversation {conversation_id}")
                # Generate a fallback title based on the first user message
                first_msg = user_messages[0][:30] + "..." if len(user_messages[0]) > 30 else user_messages[0]
                return f"Chat about {first_msg}"

            # Apply additional summary text processing to ensure clean title
            title = self.response_processor.process_summary_text(raw_title)

            # Clean up the title
            title = title.replace('"', '').replace("'", "")
            if len(title) > 50:
                title = title[:47] + "..."

            # Validate the title is meaningful (not just "New Conversation" or similar)
            if title.lower() in ["new conversation", "conversation", "chat", "new chat", ""]:
                logger.warning(f"Generic title generated for conversation {conversation_id}: '{title}'")
                # Generate a fallback title based on the first user message
                first_msg = user_messages[0][:30] + "..." if len(user_messages[0]) > 30 else user_messages[0]
                return f"Chat about {first_msg}"

            logger.info(f"Successfully generated title for conversation {conversation_id}: '{title}'")
            return title

        except Exception as e:
            logger.error(f"Failed to generate conversation title for conversation {conversation_id}: {e}")

            # Check for token limit errors specifically
            error_str = str(e).lower()
            if "token" in error_str or "limit" in error_str or "exceed" in error_str or "truncated" in error_str:
                logger.warning(f"Possible token limit issue when generating title for conversation {conversation_id}: {e}")

            # Generate a fallback title based on the first user message
            try:
                first_msg = user_messages[0][:30] + "..." if len(user_messages[0]) > 30 else user_messages[0]
                fallback_title = f"Chat about {first_msg}"
                logger.info(f"Using fallback title for conversation {conversation_id}: '{fallback_title}'")
                return fallback_title
            except Exception as fallback_error:
                logger.error(f"Failed to generate fallback title: {fallback_error}")
                return None

    async def create_conversation_summary(self, conversation_id: int) -> Optional[ConversationSummary]:
        """Create a semantic summary of the conversation using LLM"""
        messages = self.get_conversation_messages(conversation_id)

        if len(messages) < 5:  # Not enough messages to summarize
            return None

        # Check if summary already exists
        existing_summary = self.db.query(ConversationSummary).filter(
            ConversationSummary.conversation_id == conversation_id
        ).first()

        if existing_summary and existing_summary.message_count >= len(messages):
            return existing_summary  # Summary is up to date

        # Generate semantic summary and keywords
        summary_text, keywords = await self._generate_summary_text(messages)

        # Calculate priority score
        try:
            from search_manager import SearchManager
            search_manager = SearchManager(self.db)
            priority_score = search_manager.calculate_conversation_priority(conversation_id)
        except Exception as e:
            logger.warning(f"Could not calculate priority score: {e}")
            priority_score = 0.5  # Default priority

        if existing_summary:
            existing_summary.summary = summary_text
            existing_summary.keywords = keywords
            existing_summary.message_count = len(messages)
            existing_summary.priority_score = priority_score
            existing_summary.updated_at = datetime.now()
            self.db.commit()
            return existing_summary
        else:
            summary = ConversationSummary(
                conversation_id=conversation_id,
                summary=summary_text,
                keywords=keywords,
                message_count=len(messages),
                priority_score=priority_score
            )
            self.db.add(summary)
            self.db.commit()
            self.db.refresh(summary)
            return summary

    async def _generate_summary_text(self, messages: List[Message]) -> Tuple[str, str]:
        """Generate semantic summary and keywords using LLM"""
        try:
            # Prepare conversation text for summarization
            conversation_text = self._prepare_conversation_for_summary(messages)

            # Generate summary using LLM
            summary_context = [
                {
                    "role": "system",
                    "content": """Create a concise, searchable summary of this conversation. Focus on:
1. Main topics discussed
2. Key questions asked
3. Important information shared
4. Any decisions made or plans discussed
5. Problems solved or issues addressed

Keep the summary under 200 words and make it useful for future reference."""
                },
                {
                    "role": "user",
                    "content": f"Summarize this conversation:\\n\\n{conversation_text}"
                }
            ]

            summary_response = await lmstudio_client.chat_completion(
                messages=summary_context,
                temperature=0.3,
                max_tokens=200
            )

            # Process the summary response to remove thinking tags
            processed_summary_response = self.response_processor.process_chat_response(summary_response)
            raw_summary = processed_summary_response["choices"][0]["message"]["content"].strip()
            summary = self.response_processor.process_summary_text(raw_summary)

            # Generate keywords
            keywords_context = [
                {
                    "role": "system",
                    "content": "Extract 10-15 searchable keywords from this conversation summary. Return as comma-separated list. Include topics, concepts, actions, and important terms."
                },
                {
                    "role": "user",
                    "content": f"Extract keywords from: {summary}"
                }
            ]

            keywords_response = await lmstudio_client.chat_completion(
                messages=keywords_context,
                temperature=0.1,
                max_tokens=100
            )

            # Process the keywords response to remove thinking tags
            processed_keywords_response = self.response_processor.process_chat_response(keywords_response)
            raw_keywords = processed_keywords_response["choices"][0]["message"]["content"].strip()
            keywords = self.response_processor.process_summary_text(raw_keywords)

            return summary, keywords

        except Exception as e:
            logger.error(f"LLM summary generation failed: {e}")
            # Fallback to simple summary
            return self._generate_simple_summary(messages), self._extract_simple_keywords(messages)

    def _prepare_conversation_for_summary(self, messages: List[Message]) -> str:
        """Prepare conversation text for LLM summarization"""
        # Get user and assistant messages only (skip system)
        relevant_messages = [msg for msg in messages if msg.role in [MessageRole.USER, MessageRole.ASSISTANT]]

        # Limit to last 20 messages for context
        if len(relevant_messages) > 20:
            relevant_messages = relevant_messages[-20:]

        conversation_parts = []
        for msg in relevant_messages:
            role_label = "User" if msg.role == MessageRole.USER else "Assistant"

            # Remove thinking tags from content before summarization
            content = self.response_processor.process_summary_text(msg.content)

            # Truncate very long messages
            content = content[:500] + "..." if len(content) > 500 else content
            conversation_parts.append(f"{role_label}: {content}")

        return "\\n\\n".join(conversation_parts)

    def _generate_simple_summary(self, messages: List[Message]) -> str:
        """Generate simple summary as fallback"""
        summary_parts = []

        user_messages = [msg for msg in messages if msg.role == MessageRole.USER]
        assistant_messages = [msg for msg in messages if msg.role == MessageRole.ASSISTANT]

        if user_messages:
            summary_parts.append(f"User discussed: {len(user_messages)} topics")

            # Extract first and last user messages for context (remove thinking tags)
            if len(user_messages) > 0:
                first_content = self.response_processor.process_summary_text(user_messages[0].content)
                first_msg = first_content[:100] + "..." if len(first_content) > 100 else first_content
                summary_parts.append(f"Started with: {first_msg}")

            if len(user_messages) > 1:
                last_content = self.response_processor.process_summary_text(user_messages[-1].content)
                last_msg = last_content[:100] + "..." if len(last_content) > 100 else last_content
                summary_parts.append(f"Ended with: {last_msg}")

        summary_parts.append(f"Total messages: {len(messages)}")
        summary_parts.append(f"Duration: {(messages[-1].timestamp - messages[0].timestamp).total_seconds() / 60:.1f} minutes")

        return " | ".join(summary_parts)

    def _extract_simple_keywords(self, messages: List[Message]) -> str:
        """Extract simple keywords as fallback"""
        import re

        # Combine all message content (with thinking tags removed)
        cleaned_messages = [
            self.response_processor.process_summary_text(msg.content)
            for msg in messages if msg.role == MessageRole.USER
        ]
        all_text = " ".join(cleaned_messages)

        # Extract words (3+ characters)
        words = re.findall(r'\\b[a-zA-Z]{3,}\\b', all_text.lower())

        # Remove common stop words
        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by', 'this', 'that', 'are', 'is', 'was', 'were', 'have', 'has', 'had', 'will', 'would', 'could', 'should', 'can', 'could', 'may', 'might', 'must', 'shall', 'should', 'will', 'would'}

        keywords = [word for word in words if word not in stop_words]

        # Get most frequent keywords
        word_counts = Counter(keywords)
        top_keywords = [word for word, count in word_counts.most_common(15)]

        return ", ".join(top_keywords)

    async def cleanup_old_conversations(self, user_id: int, days_old: int = 30):
        """Clean up old inactive conversations"""
        cutoff_date = datetime.now() - timedelta(days=days_old)

        old_conversations = self.db.query(Conversation).filter(
            and_(
                Conversation.user_id == user_id,
                Conversation.updated_at < cutoff_date,
                Conversation.is_active == True
            )
        ).all()

        for conversation in old_conversations:
            # Create summary before deactivating
            try:
                await self.create_conversation_summary(conversation.id)
            except Exception as e:
                logger.warning(f"Could not create summary for conversation {conversation.id}: {e}")
            conversation.is_active = False

        if old_conversations:
            logger.info(f"Archived {len(old_conversations)} old conversations for user {user_id}")
            self.db.commit()
