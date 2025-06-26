"""
Conversation Management Service
"""
import logging
from typing import List, Optional, Dict, Any
from sqlalchemy.orm import Session
from sqlalchemy import desc, and_
from datetime import datetime, timedelta
from models import Conversation, Message, User, ConversationSummary
from schemas import ConversationCreate, MessageCreate, MessageRole
from memory_manager import MemoryManager
from lmstudio_client import lmstudio_client
import asyncio

logger = logging.getLogger(__name__)


class ConversationManager:
    """Manages conversations and message history"""

    def __init__(self, db: Session):
        self.db = db
        self.memory_manager = MemoryManager(db)

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
            message.model_used = metadata.get("model_used")
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

    def build_conversation_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 20
    ) -> List[Dict[str, str]]:
        """Build conversation context for LLM"""
        # Get user memory context
        memory_context = self.memory_manager.get_memory_context(user_id)

        # Get recent messages
        messages = self.get_recent_messages(conversation_id, max_messages)

        # Build context array
        context = []

        # Add system context with user memory
        if memory_context:
            system_message = f"""You are a helpful AI assistant. Here's what you know about the user:

{memory_context}

Please use this information to provide personalized and relevant responses. Be natural and don't explicitly mention that you're using stored information unless relevant to the conversation."""
            context.append({"role": "system", "content": system_message})
        else:
            context.append({"role": "system", "content": "You are a helpful AI assistant."})

        # Add conversation messages
        for message in messages:
            context.append({
                "role": message.role,
                "content": message.content
            })

        return context

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
            return None

        # Get first few user messages for context
        user_messages = [msg.content for msg in messages if msg.role == MessageRole.USER][:3]

        if not user_messages:
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
            response = await lmstudio_client.chat_completion(
                messages=context,
                temperature=0.3,
                max_tokens=50
            )

            title = response["choices"][0]["message"]["content"].strip()

            # Clean up the title
            title = title.replace('"', '').replace("'", "")
            if len(title) > 50:
                title = title[:47] + "..."

            return title
        except Exception as e:
            logger.error(f"Failed to generate conversation title: {e}")
            return None

    def create_conversation_summary(self, conversation_id: int) -> Optional[ConversationSummary]:
        """Create a summary of the conversation"""
        messages = self.get_conversation_messages(conversation_id)

        if len(messages) < 5:  # Not enough messages to summarize
            return None

        # Check if summary already exists
        existing_summary = self.db.query(ConversationSummary).filter(
            ConversationSummary.conversation_id == conversation_id
        ).first()

        if existing_summary and existing_summary.message_count >= len(messages):
            return existing_summary  # Summary is up to date

        # Create summary content
        summary_text = self._generate_summary_text(messages)

        if existing_summary:
            existing_summary.summary = summary_text
            existing_summary.message_count = len(messages)
            existing_summary.created_at = datetime.now()
            self.db.commit()
            return existing_summary
        else:
            summary = ConversationSummary(
                conversation_id=conversation_id,
                summary=summary_text,
                message_count=len(messages)
            )
            self.db.add(summary)
            self.db.commit()
            self.db.refresh(summary)
            return summary

    def _generate_summary_text(self, messages: List[Message]) -> str:
        """Generate summary text from messages"""
        # Simple extractive summary - take key points
        summary_parts = []

        user_messages = [msg for msg in messages if msg.role == MessageRole.USER]
        assistant_messages = [msg for msg in messages if msg.role == MessageRole.ASSISTANT]

        if user_messages:
            summary_parts.append(f"User discussed: {len(user_messages)} topics")

            # Extract first and last user messages for context
            if len(user_messages) > 0:
                first_msg = user_messages[0].content[:100] + "..." if len(user_messages[0].content) > 100 else user_messages[0].content
                summary_parts.append(f"Started with: {first_msg}")

            if len(user_messages) > 1:
                last_msg = user_messages[-1].content[:100] + "..." if len(user_messages[-1].content) > 100 else user_messages[-1].content
                summary_parts.append(f"Ended with: {last_msg}")

        summary_parts.append(f"Total messages: {len(messages)}")
        summary_parts.append(f"Duration: {(messages[-1].timestamp - messages[0].timestamp).total_seconds() / 60:.1f} minutes")

        return " | ".join(summary_parts)

    def cleanup_old_conversations(self, user_id: int, days_old: int = 30):
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
            self.create_conversation_summary(conversation.id)
            conversation.is_active = False

        if old_conversations:
            logger.info(f"Archived {len(old_conversations)} old conversations for user {user_id}")
            self.db.commit()
