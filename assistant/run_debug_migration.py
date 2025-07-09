#!/usr/bin/env python
"""
Run debug columns migration
"""
import sys
import os
sys.path.append('/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')

from migrate_debug_columns import migrate_debug_columns

if __name__ == "__main__":
    try:
        migrate_debug_columns()
        print("Debug columns migration completed successfully!")
    except Exception as e:
        print(f"Debug columns migration failed: {e}")
        sys.exit(1)