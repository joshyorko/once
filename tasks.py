import os

from invoke import task


@task
def build(c):
    """Build the local Once binaries."""
    os.environ["CGO_ENABLED"] = "0"
    c.run("make build")


@task
def test(c):
    """Run the Once unit tests."""
    os.environ["CGO_ENABLED"] = "0"
    c.run("make test")


@task
def integration(c):
    """Run the Go Docker integration tests."""
    os.environ["CGO_ENABLED"] = "0"
    c.run("make integration")


@task
def install(c):
    """Build and install Once into the configured prefix."""
    os.environ["CGO_ENABLED"] = "0"
    c.run("make install")


@task(pre=[build])
def robot(c):
    """Run black-box acceptance tests against the built Once binary."""
    c.run("python -m robot -L DEBUG -d developer/tmp/robot robot_tests")
