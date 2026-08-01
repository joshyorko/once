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


@task(name="robotSmoke", pre=[build])
def robot_smoke(c):
    """Run the fast black-box acceptance subset."""
    _run_robot(c, "smoke", "developer/tmp/robot/smoke")


@task(pre=[test, build])
def robot(c):
    """Run all container-safe black-box acceptance tests."""
    _run_robot(c, "acceptance", "developer/tmp/robot/acceptance")


def _run_robot(c, tag, output_dir):
    c.run(f"python -m robot -L DEBUG --include {tag} -d {output_dir} robot_tests")
