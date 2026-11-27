# Security policy

## Supported versions

The latest `v0.1.x` tag receives fixes. The project is pre-1.0, so no older series is
maintained.

## Reporting a vulnerability

Please do **not** open a public issue. Use GitHub's private vulnerability reporting:
<https://github.com/sixnineone/meui/security/advisories/new>.

Include the version, a minimal reproduction, and the impact you see. This is a UI toolkit:
the highest-risk areas are the QML parser and id resolver (untrusted scene files) and the
platform bindings through purego. Reports are acknowledged as soon as possible; there is no
bug bounty.
