#!/usr/bin/env python3

"""
AI Assistant CLI Tool
Provides command-line interface for common operations
"""

import argparse
import asyncio
import sys
from pathlib import Path

# Add current directory to path
sys.path.append(str(Path(__file__).parent))

from database import get_db_session, init_database
from dev_utils import (
    create_sample_data, analyze_memory_patterns, simulate_conversation,
    export_user_data, cleanup_test_data, memory_consolidation_task
)
from models import User


def main():
    parser = argparse.ArgumentParser(description='AI Assistant CLI Tool')
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Init command
    subparsers.add_parser('init', help='Initialize database')
    
    # Sample data command
    subparsers.add_parser('sample-data', help='Create sample data for development')
    
    # Analysis command
    subparsers.add_parser('analyze', help='Analyze memory patterns')
    
    # Simulation command
    subparsers.add_parser('simulate', help='Simulate conversation for testing')
    
    # Export command
    export_parser = subparsers.add_parser('export', help='Export user data')
    export_parser.add_argument('user_id', type=int, help='User ID to export')
    export_parser.add_argument('--output', '-o', help='Output filename')
    
    # Cleanup command
    subparsers.add_parser('cleanup', help='Clean up test data')
    
    # Consolidate command
    subparsers.add_parser('consolidate', help='Run memory consolidation')
    
    # List users command
    subparsers.add_parser('users', help='List all users')
    
    # Server command
    server_parser = subparsers.add_parser('serve', help='Start the API server')
    server_parser.add_argument('--host', default='0.0.0.0', help='Host to bind to')
    server_parser.add_argument('--port', type=int, default=8000, help='Port to bind to')
    server_parser.add_argument('--reload', action='store_true', help='Enable auto-reload')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    # Execute command
    if args.command == 'init':
        print("🗄️  Initializing database...")
        init_database()
        print("✅ Database initialized")
    
    elif args.command == 'sample-data':
        create_sample_data()
    
    elif args.command == 'analyze':
        analyze_memory_patterns()
    
    elif args.command == 'simulate':
        simulate_conversation()
    
    elif args.command == 'export':
        if args.output:
            export_user_data(args.user_id, args.output)
        else:
            export_user_data(args.user_id)
    
    elif args.command == 'cleanup':
        cleanup_test_data()
    
    elif args.command == 'consolidate':
        memory_consolidation_task()
    
    elif args.command == 'users':
        with get_db_session() as db:
            users = db.query(User).all()
            if users:
                print("👥 Users:")
                for user in users:
                    print(f"   {user.id}: {user.username} ({user.email or 'no email'})")
            else:
                print("No users found")
    
    elif args.command == 'serve':
        import uvicorn
        from main import app
        
        print(f"🚀 Starting server on {args.host}:{args.port}")
        uvicorn.run(app, host=args.host, port=args.port, reload=args.reload)


if __name__ == '__main__':
    main()
