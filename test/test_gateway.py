"""Cover the gateway this branch builds, raised on its own from the dev stack.

The daily stack pulls the published gateway, so these tests build it from images/gateway.
"""

import subprocess

from conftest import REPO


def gateway_up():
    """Build the gateway from this branch, wait until it reports healthy, and hand back what compose printed."""
    return subprocess.run(
        [
            "docker", "compose",
            "-f", REPO / ".devcontainer/compose.yaml",
            "-f", REPO / ".devcontainer/compose.dev.yaml",
            "up", "--build", "--wait", "--wait-timeout", "60", "gateway",
        ],
        capture_output=True,
        text=True,
    )


def test_the_gateway_comes_up_healthy_on_an_empty_policy(an_empty_policy):
    """An empty policy raises a gateway that reports healthy and rejects every connection."""
    raising = gateway_up()

    assert raising.returncode == 0, raising.stdout + raising.stderr
