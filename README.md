<p align="center">
  <img src=".github/assets/logo.png" alt="o3s logo" width="170">
</p>

<h1 align="center">o3s</h1>

<p align="center">
  <b>Everything ready. Everything parallel. Everything locked down.</b>
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/architecture-dark.excalidraw.svg">
    <img src=".github/assets/architecture.excalidraw.svg" alt="Agents work side by side in the cage alongside predefined and self-built features; every connection they open goes through the gateway, which holds the allowlist and the secrets, and is either allowed out to the internet or denied" width="100%">
  </picture>
</p>

If you are not willing to trade security for productivity, o3s is a match. Let an agent generate every line of code, and review every line by hand or none at all. Either way, which hosts your agents reach and which secrets they can read stays yours to decide.

We created o3s to keep pace: trusted environments, several projects at once, and several features inside each. AI will not be the death of good engineering. It is the golden era of agentic software engineers. This is how we build with it and still stand behind every line that ships.

## Commands

Run `o3s` alone for a menu that leads through each of them.

**firewall**
```sh
# Allows the hosts the cage may reach
o3s firewall add github.com --port 22 --port 443
```

**secret**
```sh
# Lets the cage use a secret while its value stays on the host
echo "$EXAMPLE_TOKEN" | o3s secret add EXAMPLE_TOKEN
```

**store**
```sh
# Selects where the cage's tools come from, curated, reproducible and secure by default
o3s store add ghcr.io/devcontainers/features
```

**feature**
```sh
# Installs a tool with the hosts and env it needs
o3s feature add ghcr.io/hansehart/devcontainer-features/claude-code
```

**cage**
```sh
# Opens the home's workspace in VS Code inside the cage, or a shell inside it
o3s cage ide
o3s cage shell
```

**update**
```sh
# Brings the binary, its images, the stack and your features forward
o3s update
```

## Getting started

**Prerequisites:** Linux on amd64 or arm64, [Docker](https://docs.docker.com/engine/install/) with Compose, [VS Code](https://code.visualstudio.com/download) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers), and tar and xz.

1. Install o3s.
   ```sh
   curl -fsSL https://github.com/hansehart/o3s/releases/latest/download/install.sh | sh
   ```
2. Create your home.
   ```sh
   mkdir ~/o3s && cd ~/o3s && o3s init
   ```
3. Open it in the cage.
   ```sh
   # VS Code
   o3s cage ide
   ```
   or
   ```sh
   # Terminal
   o3s cage shell
   ```

> [!WARNING]
> Only `~/features`, `~/o3s`, and `~/projects` survive a rebuild in the cage. Keep your work in `~/projects` and add each project to the workspace.

## What you get

### Security

You keep the whole policy in one file, `o3s.json`, and change it with `o3s firewall`:

```json
{
  "hosts": [
    { "name": "github.com", "ports": [22, 443], "by": ["user"] },
    { "name": "api.example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "inject": ["header"], "by": ["user"] }
  ]
}
```

A host is a domain, a wildcard like `*.docker.io` that covers every subdomain, an IPv4 address or an IPv4 range. Every host names its ports, and a host with a secret carries it in its requests' headers, or in the query or body it names. Each change restarts the gateway and the proxy, so the cage reaches the new policy at once. The value of a secret stays on the host in `~/.config/o3s/secrets.env`. The cage sees only a stand-in, which the proxy swaps for the value on the way out.

### Productivity

You list every project in one file, `~/o3s/o3s.code-workspace`, one folder per repository or worktree:

```json
"folders": [
  { "name": "devcontainer-features", "path": "/home/ubuntu/projects/devcontainer-features" },
  { "name": "infra",                 "path": "/home/ubuntu/projects/infra" },
  { "name": "infra.worktrees",       "path": "/home/ubuntu/projects/infra.worktrees" },
  { "name": "malea-backend",         "path": "/home/ubuntu/projects/malea-backend" },
  { "name": "malea-docs",            "path": "/home/ubuntu/projects/malea-docs" },
  { "name": "malea-graph",           "path": "/home/ubuntu/projects/malea-graph" },
  { "name": "malea-web",             "path": "/home/ubuntu/projects/malea-web" },
  { "name": "works-on-my-machine",   "path": "/home/ubuntu/projects/works-on-my-machine" }
]
```

Open it and every repository lands in one source-control view, worktrees nested under the repository they came from:

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/workspace-dark.png">
    <img src=".github/assets/workspace.png" alt="Several repositories in one o3s source-control view, each on its own branch, with worktrees nested under the repository they came from" width="100%">
  </picture>
</p>

Add the agents and tools you need as features, then run whatever agent you like, several at once, each on its own branch. They work in parallel without colliding, and everything lands in one place for you to review.

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/agents-dark.png">
    <img src=".github/assets/agents.png" alt="Claude Code and Codex CLI running in four panes at once inside one o3s dev container" width="100%">
  </picture>
</p>

### Velocity

Break early, fix earlier, ship earliest. Rootless Docker and minikube run inside the cage, so an agent can stand up your whole stack, break it, and stand it up again without touching anything real. The stack behaves like production, and the loop from failure to the next attempt stays short.

## References

> [!TIP]
> Is o3s worth your limited time? Decide on your security model:
>
> **My agent is actively malicious and tries to escape**\
> ❌ No, a VM boundary is stronger than containers.
>
> **My long-leash-but-not-weaponized agent needs deterministic boundaries**\
> ✅ Yes, the architecture delivers security coupled with convenience.

See the [comparison](https://github.com/Hansehart/o3s/wiki/Comparison) for others working on the same problem.

Something here you would do differently? Open an issue, send a pull request, or just get in touch. And if one of the others fits you better, take it, as long as you keep both your security and your productivity. You are always welcome back.

## What the name means

<table align="center" width="80%">
  <tr>
    <td width="50%" valign="top" align="center">
      <img src="https://img.shields.io/badge/Open-29b6ee?style=for-the-badge" alt="Open">
      <p>Read every line, shape it however you like, and make the whole setup truly your own to trust and extend.</p>
    </td>
    <td width="50%" valign="top" align="center">
      <img src="https://img.shields.io/badge/Secure-29b6ee?style=for-the-badge" alt="Secure">
      <p>Give AI agents and untrusted code a long leash while you control the network they reach and the secrets they read.</p>
    </td>
  </tr>
</table>

<table align="center" width="80%">
  <tr>
    <td width="50%" valign="top" align="center">
      <img src="https://img.shields.io/badge/Software-29b6ee?style=for-the-badge" alt="Software">
      <p>Built for engineers and their agents to take your whole stack from prototype to production with minimal friction.</p>
    </td>
    <td width="50%" valign="top" align="center">
      <img src="https://img.shields.io/badge/Suite-29b6ee?style=for-the-badge" alt="Suite">
      <p>Agents, containers, languages, and all your projects, wired together into one platform you just open and use.</p>
    </td>
  </tr>
</table>

<p align="center">
  open secure software suite, or osss, shortened to o3s.
</p>
