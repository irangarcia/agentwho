# Security policy

## Supported versions

Security fixes are made for the latest released version of AgentWho. Upgrade to the latest release before reporting an issue that may already have been resolved.

## Report a vulnerability

Please report security vulnerabilities privately through GitHub:

1. Open the repository's **Security** tab.
2. Select **Report a vulnerability**.
3. Describe the issue and its potential impact.

Do not open a public issue for an undisclosed vulnerability.

Include, when relevant:

- the AgentWho, operating system, shell, Claude Code, and Codex CLI versions;
- the smallest safe reproduction you can provide;
- whether the issue affects profile isolation, mismatch enforcement, command discovery, shell integration, or filesystem permissions;
- suggested mitigations, if known.

Never include authentication tokens, credential-file contents, private source code, or other secrets. Use temporary directories, fake executables, and test repositories whenever possible.

## What to expect

We aim to acknowledge a report within seven days, keep the reporter informed while it is investigated, and coordinate disclosure after a fix is available. Please allow time for investigation before sharing the issue publicly.

## Security boundary

AgentWho must not read, parse, copy, migrate, or display Claude or Codex credentials. It isolates official CLI state through profile-specific configuration directories and delegates authentication to the official CLIs.

Reports about bypassing profile mismatch protection, escaping profile directories, replacing unmanaged executables, unsafe shell integration, or exposing credential material are especially important.
