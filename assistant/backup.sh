#!/bin/bash
# Backup script for wikillm assistant

BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup important files
cp assistant.db "$BACKUP_DIR/assistant.db" 2>/dev/null || echo "No database to backup"
cp memory_manager.py "$BACKUP_DIR/memory_manager.py.backup"
cp main.py "$BACKUP_DIR/main.py.backup"
cp conversation_manager.py "$BACKUP_DIR/conversation_manager.py.backup"
cp config.py "$BACKUP_DIR/config.py.backup"

echo "Backup created in $BACKUP_DIR"
