# Configuration

Normal AgentWho use should go through CLI commands. The YAML file is an advanced escape hatch for inspection, backup, and carefully reviewed automation.

AgentWho does not read `.agentwho.yaml` or any repository-local configuration file.

## Filesystem layout

AgentWho respects XDG environment variables.

Configuration:

```text
${XDG_CONFIG_HOME:-~/.config}/agentwho/config.yaml
```

Application data:

```text
${XDG_DATA_HOME:-~/.local/share}/agentwho/
├── bin/
│   ├── claude
│   └── codex
└── profiles/
    ├── personal/
    │   ├── claude/
    │   └── codex/
    └── work/
        ├── claude/
        └── codex/
```

Configuration and data directories use owner-only permissions. Each agent owns its own subdirectory inside each profile.

## Complete schema

```yaml
version: 1
defaults:
  profile: personal
  enforcement: confirm
profiles:
  personal:
    kind: personal
  work:
    kind: work
rules:
  - match:
      git_remote: github.com/acme/backend
    profile: work
    enforcement: block
  - match:
      git_organization: github.com/acme
    profile: work
    enforcement: block
  - match:
      path: ~/code/personal
    profile: personal
    enforcement: confirm
```

### `version`

Required integer. The only supported value is `1`.

### `defaults.profile`

Required profile name. It must refer to a key in `profiles`. AgentWho uses it when no repository, organization, or path rule matches.

### `defaults.enforcement`

Required safety mode. Supported values:

- `block` — refuse every mismatch;
- `confirm` — ask interactively, but refuse in non-interactive use.

The CLI calls this **safety mode**. Configuration version 1 retains the YAML field name `enforcement`.

### `profiles`

Required mapping with at least one entry. Profile names:

- contain only lowercase letters, numbers, and single hyphens between groups;
- begin with a letter or number;
- contain at most 63 characters;
- cannot contain whitespace, uppercase characters, separators, traversal sequences, or shell metacharacters.

Valid examples:

```text
personal
work
client-acme
work-2
```

Every profile requires one kind:

- `personal`;
- `work`.

Kinds allow mismatch warnings to explain the exposure direction. They do not grant permissions or inspect the underlying account.

Create profiles through the CLI:

```sh
agentwho profile add work --kind work
agentwho profile add client-acme --kind work
```

### `rules`

Optional list of bindings. Every rule requires exactly one matcher, an existing profile, and a safety mode.

Supported matchers:

- `git_remote` — exact normalized repository remote;
- `git_organization` — normalized host plus top-level namespace;
- `path` — absolute canonical directory tree.

Equivalent duplicate matchers are rejected even when they reference different profiles. Use `agentwho unbind` before replacing an existing binding.

Create rules through the CLI:

```sh
agentwho bind work --repo --safety-mode block
agentwho bind work --organization --safety-mode block
agentwho bind personal --path ~/projects/personal --safety-mode confirm
```

## Resolution precedence

The highest-specificity matching rule wins:

1. exact Git repository remote;
2. Git organization or top-level namespace;
3. longest containing directory path;
4. default profile.

The most deeply nested path wins among multiple matching path rules. A repository rule always outranks an organization or path rule. An organization rule always outranks a path rule.

Inspect resolution and its explanation:

```sh
agentwho status
agentwho rules
```

## Git remote normalization

AgentWho supports SCP-style SSH, URI-style SSH, HTTPS, GitLab, and self-hosted Git servers.

These values all normalize to `github.com/acme/backend`:

```text
git@github.com:acme/backend.git
https://github.com/acme/backend.git
ssh://git@github.com/acme/backend.git
```

Normalization:

- lowercases the host;
- preserves an explicit non-default port;
- removes leading and trailing path separators;
- removes one trailing `.git` suffix;
- rejects empty or traversal-like paths.

For `gitlab.example.com/platform/payments/api`, the organization matcher is `gitlab.example.com/platform`. Deeper subgroup distinctions should use an exact-repository or path rule.

Only `origin` is considered. Other remotes do not participate in resolution.

## Path normalization

When a path binding is added, AgentWho:

1. expands `~` using the current home directory;
2. resolves a relative CLI argument against the current directory;
3. converts it to an absolute clean path;
4. resolves symlinks when the target exists.

Path rules apply recursively while preserving directory boundaries. A rule for `/code/acme` matches `/code/acme/backend`, but not `/code/acme-other`.

The same canonicalization is applied to the current directory during resolution.

## Atomic updates and validation

Configuration parsing rejects:

- unknown YAML fields;
- multiple YAML documents;
- unsupported versions;
- empty profile maps;
- invalid names or kinds;
- missing default profiles;
- unsupported safety modes;
- rules with zero or multiple matchers;
- rules referencing unknown profiles;
- duplicate equivalent matchers;
- non-absolute path rules after normalization.

Errors identify the invalid field. CLI updates are written to a temporary file in the configuration directory, assigned restrictive permissions, synchronized, and atomically renamed over the previous file.

## Explicit selection and bypass

With no override, the current profile always equals the profile AgentWho resolves for the context.

Select a profile for the current shell:

```sh
agentwho use personal
```

Return to repository-based selection:

```sh
agentwho use --auto
```

`agentwho use` is intentionally shell-local. It requires the function generated by `agentwho shell init`, because an ordinary child process cannot modify its parent shell. If integration is not active, AgentWho prints the exact initialization command to run.

The selected profile must exist. A conflict with the expected profile invokes the binding's safety mode.

### Advanced environment interface

`agentwho use` uses `AGENTWHO_PROFILE` inside the current shell. Scripts that need a single non-interactive explicit selection may use the lower-level interface directly:

```sh
AGENTWHO_PROFILE=personal claude
```

This is an automation interface, not the recommended interactive workflow. It performs the same validation and mismatch checks.

### Emergency force override

`AGENTWHO_FORCE=1` never acts silently.

Interactive execution requires an explicit profile mismatch plus typing that selected profile name when prompted:

```sh
agentwho use personal
AGENTWHO_FORCE=1 claude
```

Non-interactive execution requires both variables:

```sh
AGENTWHO_PROFILE=personal AGENTWHO_FORCE=1 claude
```

AgentWho prints a bypass warning to stderr in either case.

After enforcement, AgentWho removes `AGENTWHO_FORCE` from the agent process and sets `AGENTWHO_PROFILE` to the profile actually used.

## Agent environments

AgentWho preserves the current environment and replaces only the supported agent's account directory:

| Agent | Environment variable | Profile path |
| --- | --- | --- |
| Claude Code | `CLAUDE_CONFIG_DIR` | `profiles/<profile>/claude` |
| Codex CLI | `CODEX_HOME` | `profiles/<profile>/codex` |

An inherited value for the same agent variable is removed before the selected profile path is added. AgentWho never reads files in those directories.

## Stable JSON interfaces

JSON output contains no credentials or complete environments.

### Current profile

```sh
agentwho current --json
```

```json
{
  "version": 1,
  "profile": "work",
  "expected_profile": "work",
  "source": "automatic",
  "mismatch": false
}
```

`source` is either `automatic` or `explicit`.

### Status

```sh
agentwho status --json
```

```json
{
  "directory": "/Users/example/projects/acme/backend",
  "git_root": "/Users/example/projects/acme/backend",
  "git_remote": "github.com/acme/backend",
  "matched_rule": {
    "match": {
      "git_organization": "github.com/acme"
    },
    "profile": "work",
    "safety_mode": "block",
    "enforcement": "block"
  },
  "specificity": "git organization",
  "expected_profile": "work",
  "current_profile": "work",
  "safety_mode": "block",
  "selected_profile": "work",
  "enforcement": "block",
  "claude_shim_installed": true,
  "codex_shim_installed": true,
  "automatic_selection_active": true,
  "mismatch": false,
  "status": "OK"
}
```

Git and matched-rule fields are omitted when unavailable. `selected_profile` and `enforcement` remain aliases for version-1 script compatibility.

### Rules

```sh
agentwho rules --json
```

```json
[
  {
    "matcher_type": "git_organization",
    "matcher_value": "github.com/acme",
    "profile": "work",
    "safety_mode": "block",
    "enforcement": "block",
    "specificity": "git organization"
  }
]
```

### Profiles

```sh
agentwho profile list --json
```

```json
[
  {
    "name": "personal",
    "kind": "personal",
    "claude": "authenticated",
    "codex": "not authenticated"
  }
]
```

Agent status values are `authenticated`, `not authenticated`, `unavailable`, or `unknown`.

### Prompt

```sh
agentwho prompt --json
```

```json
{
  "initialized": true,
  "profile": "work",
  "mismatch": false,
  "text": "[agent:work]"
}
```

When AgentWho is uninitialized, prompt JSON contains `initialized: false` and omits profile text.

## Related documentation

- [Architecture](architecture.md)
- [VS Code integration](vscode.md)
- [Known limitations](limitations.md)
- [README](../README.md)
