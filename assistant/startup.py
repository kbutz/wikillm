#!/usr/bin/env python3
"""
Startup script for AI Assistant with Cross-Conversation Search
"""
import sys
import os
import subprocess
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def run_migration():
    """Run database migration"""
    try:
        result = subprocess.run([
            sys.executable, 
            'simple_migration.py'
        ], capture_output=True, text=True, cwd='/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')
        
        if result.returncode == 0:
            logger.info("✅ Database migration completed successfully")
            if result.stdout:
                logger.info(f"Migration output: {result.stdout}")
            return True
        else:
            logger.error(f"❌ Migration failed with return code {result.returncode}")
            if result.stderr:
                logger.error(f"Migration error: {result.stderr}")
            return False
    except Exception as e:
        logger.error(f"❌ Failed to run migration: {e}")
        return False

def test_application():
    """Test that the application components work"""
    try:
        result = subprocess.run([
            sys.executable,
            'test_fixes.py'
        ], capture_output=True, text=True, cwd='/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')
        
        if result.returncode == 0:
            logger.info("✅ Application tests passed")
            if result.stdout:
                logger.info(f"Test output: {result.stdout}")
            return True
        else:
            logger.error(f"❌ Tests failed with return code {result.returncode}")
            if result.stderr:
                logger.error(f"Test error: {result.stderr}")
            return False
    except Exception as e:
        logger.error(f"❌ Failed to run tests: {e}")
        return False

def start_application():
    """Start the FastAPI application"""
    try:
        logger.info("🚀 Starting AI Assistant with Cross-Conversation Search...")
        
        # Change to application directory
        os.chdir('/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant')
        
        # Start the application
        subprocess.run([sys.executable, 'main.py'])
        
    except KeyboardInterrupt:
        logger.info("👋 Application stopped by user")
    except Exception as e:
        logger.error(f"❌ Failed to start application: {e}")
        return False

def main():
    """Main startup sequence"""
    print("🤖 AI Assistant Startup Script")
    print("=" * 50)
    
    # Step 1: Run migration
    print("\n📁 Step 1: Running database migration...")
    if not run_migration():
        print("❌ Migration failed. Please check the logs and try again.")
        return False
    
    # Step 2: Test application
    print("\n🧪 Step 2: Testing application components...")
    if not test_application():
        print("❌ Tests failed. Please check the logs and fix issues before starting.")
        return False
    
    # Step 3: Start application
    print("\n🚀 Step 3: Starting application...")
    print("The application will be available at: http://localhost:8000")
    print("API documentation at: http://localhost:8000/docs")
    print("Press Ctrl+C to stop the application")
    print("-" * 50)
    
    start_application()
    return True

if __name__ == "__main__":
    success = main()
    if not success:
        sys.exit(1)
