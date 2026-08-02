# Contributing to AgentWho

Thanks for helping AgentWho automatically use the correct Claude and Codex account for every project.

## Requirements

- Go 1.22 or newer;
- macOS or Linux;
- Git;
- `make`.

Real Claude Code and Codex sessions are not required for development or tests.

## Local workflow

```sh
git clone https://github.com/irangarcia/agentwho.git
cd agentwho
make test
make vet
make build
```

The executable is written to `bin/agentwho`.

Before submitting a change:

```sh
gofmt -w ./cmd ./internal ./integration
go test ./...
go vet ./...
```

Do not run formatting commands over generated or third-party files.

## Project structure

```text
cmd/agentwho/       CLI entry point
internal/agent/     Claude and Codex adapters
internal/cli/       Cobra commands and terminal presentation
internal/config/    YAML model, validation, and atomic persistence
internal/enforce/   Bidirectional mismatch protection
internal/execution/ Official executable discovery and process launch
internal/gitctx/    Git context and path normalization
internal/paths/     XDG filesystem layout
internal/resolve/   Binding precedence
internal/shell/     Shell initialization and reversible blocks
internal/shim/      Managed Claude and Codex commands
internal/tui/       Interactive arrow-key menus
integration/        End-to-end tests with fake agent executables
docs/               User and architecture documentation
```

Agent-specific behavior belongs in `internal/agent`, not Cobra handlers. Configuration and resolution logic should remain independently testable.

## Testing principles

Tests must never start a real Claude or Codex session or inspect real credential files.

Use:

- temporary XDG directories;
- temporary Git repositories;
- fake executable directories and `PATH` values;
- fake Claude and Codex programs;
- injected readers, writers, environment access, and process replacement.

Add table-driven tests for new normalization, validation, and resolution cases. Behavior changes to account enforcement should include both personal-to-work and work-to-personal directions plus interactive and non-interactive coverage.

The integration test should continue to prove the complete protected-command flow with fake executables.

## Security rules

Changes must not make AgentWho:

- read or parse credential contents;
- copy or migrate credentials;
- store tokens or API keys;
- execute repository-provided AgentWho policy;
- invoke an agent through `sh -c`;
- silently bypass mismatch protection;
- overwrite or delete an unmanaged executable;
- print complete environments or secrets;
- add telemetry or network behavior without an explicit project decision.

Treat repository paths and remotes as untrusted data.

## Pull requests

Keep pull requests focused. Include:

- a clear description of the user-visible problem;
- tests for changed behavior;
- documentation updates when commands or output change;
- `go test ./...` and `go vet ./...` results;
- screenshots or terminal output for meaningful interactive changes.

Avoid unrelated dependency additions. Prefer the Go standard library when practical and explain any new dependency.

## Documentation assets

Terminal visuals live in `docs/assets`. The checked-in `scripts/generate-demo.swift` script regenerates the current GIF and screenshots on macOS:

```sh
swift scripts/generate-demo.swift
```

Keep visuals free of real usernames, company names, tokens, and local credential paths.
