#!/usr/bin/env python3

import os
import stat

# Make scripts executable
scripts = [
    'setup_mcp.sh',
    'test_mcp.py', 
    'debug_mcp.py'
]

for script in scripts:
    if os.path.exists(script):
        # Make executable
        st = os.stat(script)
        os.chmod(script, st.st_mode | stat.S_IEXEC)
        print(f"✅ Made {script} executable")
    else:
        print(f"⚠️  {script} not found")

print("🎯 Setup complete! Run './setup_mcp.sh' to start")
