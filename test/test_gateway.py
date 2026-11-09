"""Cover the gateway this branch builds, raised on its own from the dev stack.

The daily stack pulls the published gateway, so these tests build it from images/gateway.
"""

import subprocess

from conftest import REPO

COMPOSE = [
    "docker", "compose",
    "-f", REPO / ".devcontainer/compose.yaml",
    "-f", REPO / ".devcontainer/compose.dev.yaml",
]


def gateway_up():
    """Build the gateway from this branch, wait until it reports healthy, and hand back what compose printed."""
    return subprocess.run(
        [*COMPOSE, "up", "--build", "--wait", "--wait-timeout", "60", "gateway"],
        capture_output=True,
        text=True,
    )


def gateway_exec(*command):
    """Run a command in the running gateway and hand back what it printed."""
    return subprocess.run(
        [*COMPOSE, "exec", "-T", "gateway", *command],
        capture_output=True,
        text=True,
        check=True,
    ).stdout


def test_the_gateway_comes_up_healthy_on_an_empty_policy(an_empty_policy):
    """An empty policy raises a gateway that reports healthy."""
    raising = gateway_up()

    assert raising.returncode == 0, raising.stdout + raising.stderr


def test_the_gateway_rejects_every_new_connection_on_an_empty_policy(an_empty_policy):
    """On an empty policy the gateway forwards replies alone and rejects every new connection."""
    raising = gateway_up()
    # hold the gateway to running, so the rules below come from a gateway that started
    assert raising.returncode == 0, raising.stdout + raising.stderr

    rules = gateway_exec("iptables", "-S", "FORWARD").splitlines()

    accepted = [rule for rule in rules if rule.endswith("-j ACCEPT")]
    assert all("--ctstate" in rule for rule in accepted), accepted
    assert rules[-1] == "-A FORWARD -j REJECT --reject-with icmp-admin-prohibited", rules
