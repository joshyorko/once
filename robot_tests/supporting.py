import re
import socket
import subprocess
import uuid


ANSI_ESCAPE = re.compile(
    r"\x1b(?:\][^\x07]*(?:\x07|\x1b\\)|\[[0-?]*[ -/]*[@-~])"
)


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def strip_ansi(value):
    return ANSI_ESCAPE.sub("", value)


def make_run_id():
    return uuid.uuid4().hex[:10]


def once_resources(namespace):
    commands = (
        ("docker", "ps", "-a", "--format", "{{.Names}}"),
        ("docker", "network", "ls", "--format", "{{.Name}}"),
        ("docker", "volume", "ls", "--format", "{{.Name}}"),
    )
    resources = []
    for command in commands:
        result = subprocess.run(command, check=True, capture_output=True, text=True)
        resources.extend(
            name
            for name in result.stdout.splitlines()
            if name == namespace or name.startswith(f"{namespace}-")
        )
    return sorted(set(resources))
