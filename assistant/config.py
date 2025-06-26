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
    max_user_memory_entries: int = 100
    memory_consolidation_threshold: int = 10

    # Response Settings
    default_temperature: float = 0.7
    default_max_tokens: int = 2048

    # Security
    cors_origins: list = ["*"]

    class Config:
        env_file = ".env"


settings = Settings()
