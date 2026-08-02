# VS Code integration

AgentWho protects Claude and Codex commands launched from a VS Code integrated terminal when that terminal has AgentWho shell integration active.

## Integrated terminal

Open a new VS Code terminal and run:

```sh
agentwho status
command -v claude
command -v codex
```

A working setup reports both commands as `AgentWho active`. The command paths should point into:

```text
${XDG_DATA_HOME:-~/.local/share}/agentwho/bin/
```

From that terminal, ordinary commands are protected:

```sh
claude
codex
```

## If the integrated terminal is not protected

First activate AgentWho in the current terminal:

```sh
eval "$(agentwho shell init zsh)"
agentwho status
```

Use `bash` instead of `zsh` when appropriate. For fish:

```fish
agentwho shell init fish | source
```

If that works, add the printed initialization line to the shell file loaded by VS Code or let AgentWho offer a backed-up update:

```sh
agentwho install --modify-shell
```

Then close the old integrated terminal and create a new one. `agentwho doctor` reports shell and `PATH` problems with suggested fixes.

## Graphical extension panels

Graphical Claude or Codex extension panels are **not protected in this version**. An extension may launch its own process, manage separate credentials, or bypass the terminal's `PATH`, so AgentWho cannot reliably intercept it.

AgentWho intentionally does not modify VS Code settings, extension storage, or extension credentials. Use the integrated terminal when account isolation must be enforced.

## Remote development and containers

VS Code Remote SSH, Dev Containers, and similar environments run their terminal and extensions in a different environment from the host. Install and initialize AgentWho inside the environment where `claude` or `codex` actually runs.

A host installation does not automatically protect commands inside a remote machine or container. Verify each environment independently with `agentwho status` and `agentwho doctor`.

## Future editor support

Direct protection for graphical extension panels would require an editor integration or agent-supported launch configuration. That is outside the first release; see [Known limitations](limitations.md).
