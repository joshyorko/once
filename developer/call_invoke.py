import os
import subprocess
import sys


repository_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

assert len(sys.argv) >= 2, "No task provided when calling `call_invoke.py`"
task = sys.argv[1]
raise SystemExit(subprocess.run(("invoke", task), cwd=repository_root).returncode)
