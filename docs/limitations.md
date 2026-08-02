# Known limitations

AgentWho is intentionally a focused first release.

## Platforms

- macOS and Linux are supported.
- Windows is not supported.
- Shell integration supports zsh, bash, and fish.

## Agents

- Claude Code and Codex CLI are supported.
- Other coding agents are not supported.
- Graphical Claude and Codex editor panels are not protected; integrated terminal commands can be protected. See [VS Code integration](vscode.md).

## Git and bindings

- Only the Git `origin` remote participates in repository resolution.
- Organization matching uses the first namespace segment after the host.
- Use an exact-repository or directory binding when deeper GitLab subgroup distinctions matter.
- A local repository without a usable `origin` can be bound only by directory.
- AgentWho never reads repository-local configuration.

## Authentication status

- Status depends on the official CLI exposing and maintaining its documented status command.
- A missing CLI reports `unavailable`; an unexpected or timed-out result reports `unknown`.
- AgentWho never falls back to reading credential files.

## Explicitly out of scope

The first release has no:

- daemon;
- graphical interface;
- cloud synchronization;
- telemetry;
- Git hook;
- VS Code extension;
- repository-local AgentWho file;
- organization administration;
- remote policy management;
- automatic credential copying or migration.

These boundaries keep AgentWho local, predictable, reversible, and independent of credential contents.
