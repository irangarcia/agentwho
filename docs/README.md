# AgentWho documentation

AgentWho automatically uses the correct Claude and Codex account for every project.

## Guides

- [Architecture](architecture.md) — command interception, executable discovery, account isolation, adapters, and process behavior.
- [Configuration](configuration.md) — XDG paths, YAML schema, matching rules, precedence, environment overrides, and JSON interfaces.
- [VS Code](vscode.md) — what is protected in integrated terminals and what is not protected in extension panels.
- [Known limitations](limitations.md) — supported platforms, agents, Git behavior, and intentionally omitted features.
- [Contributing](../CONTRIBUTING.md) — local development, tests, project structure, and pull requests.

## Command help

The CLI is the authoritative command reference:

```sh
agentwho help
agentwho <command> --help
```

Frequently used commands:

```sh
agentwho status
agentwho current
agentwho use work
agentwho profile list
agentwho rules
agentwho doctor
```

Return to the [README](../README.md) for installation and onboarding.
