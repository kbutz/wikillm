#!/usr/bin/env python
"""
Run database migration
"""
import sys
import os
sys.path.append('/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')

from migrate_database import migrate_database, populate_existing_summaries

if __name__ == "__main__":
    try:
        migrate_database()
        populate_existing_summaries()
        print("Migration completed successfully!")
    except Exception as e:
        print(f"Migration failed: {e}")
        sys.exit(1)
