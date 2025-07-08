"""
AI Assistant Configuration Settings
"""
from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    # Database
    database_url: str = "sqlite:///./assistant.db"

    # LMStudio Integration
    lmstudio_base_url: str = "http://localhost:1234"
    lmstudio_model: str = "local-model"
    lmstudio_timeout: int = 600  # 10 minutes to accommodate slow local LMStudio responses

    # API Settings
    api_host: str = "0.0.0.0"
    api_port: int = 8000
    api_title: str = "AI Assistant API"
    api_version: str = "1.0.0"

    # Memory Settings
    max_conversation_history: int = 50
    max_user_memory_entries: int = 200  # Increased from 100
    memory_consolidation_threshold: int = 5  # Reduced from 10 for more frequent consolidation
    
    # Search Settings
    enable_cross_conversation_search: bool = True
    auto_summarize_after_messages: int = 3  # Reduced from 5 for more frequent summaries
    max_search_results: int = 15  # Increased from 10
    priority_conversation_threshold: float = 0.2  # Reduced from 0.3 for more sensitive priority detection
    
    # Enhanced Memory Settings
    memory_extraction_confidence_threshold: float = 0.6  # Minimum confidence for storing memories
    memory_semantic_search_enabled: bool = True
    memory_cleanup_interval_days: int = 30
    fts_search_enabled: bool = True

    # Response Settings
    default_temperature: float = 0.7
    default_max_tokens: int = 2048
    
    # Memory Extraction Settings
    memory_extraction_temperature: float = 0.1  # Low temperature for consistent extraction
    memory_extraction_max_tokens: int = 500
    memory_validation_enabled: bool = True

    # Security
    cors_origins: list = ["*"]

    class Config:
        env_file = ".env"


settings = Settings()
