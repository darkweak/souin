# Security Policy

## Supported versions

Only the latest released version of Souin (and of each plugin built from this repository) receives security fixes. Please check that the issue still reproduces on the latest release, or on `master`, before reporting it.

## Reporting a vulnerability

Please **do not open a public GitHub issue** for a security vulnerability.

Report it privately, either:

- through GitHub's [private vulnerability reporting](https://github.com/darkweak/souin/security/advisories/new), or
- by email to [darkweak@protonmail.com](mailto:darkweak@protonmail.com).

Include as much of the following as you can:

- the affected version, plugin (Caddy, Traefik, …) and storage backend;
- the configuration needed to reproduce it;
- the steps or requests that trigger it, and what an attacker gains (cache poisoning, data leak between users, denial of service, …).

## What to expect

Souin is maintained on volunteer time, so there is no guaranteed response time, but reports are acknowledged as soon as possible. Once a fix is ready it is released, and the advisory is published with credit to the reporter unless you ask otherwise.

Low-severity issues that don't put users at risk can be reported as regular GitHub issues.
