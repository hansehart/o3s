"""Cover the gateway this branch builds, raised on its own from the dev stack.

The daily stack pulls the published gateway, so these tests build it from images/gateway and
start it alone, with no cage behind it.
"""

import subprocess

from conftest import REPO


def gateway_up():
    """Build the gateway from this branch and wait until it reports healthy, or report why not."""
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


def test_the_gateway_comes_up_healthy_on_a_policy_that_allows_nothing(a_policy_that_allows_nothing):
    """A policy without a single host raises a gateway that stays healthy and denies everything."""
    raising = gateway_up()

    assert raising.returncode == 0, raising.stdout + raising.stderr
