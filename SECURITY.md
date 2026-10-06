# Security Policy

## Supported Versions

Only the latest release of FiberForge receives security updates.

| Version | Supported          |
| ------- | ------------------ |
| >= 1.0.x | :white_check_mark: |
| < 1.0.0  | :x:                |

## Reporting a Vulnerability

We take the security of FiberForge and the code it generates seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

Please **do not disclose security vulnerabilities publicly** (such as via public GitHub issues, discussions, or social media) until they have been addressed and a patch is released.

Instead, please report vulnerabilities through:
- **GitHub Private Vulnerability Reporting**: Submit a private advisory via the [FiberForge Security Advisories page](https://github.com/v-pat/fiberforge/security/advisories/new).

If private vulnerability reporting is unavailable, please open a confidential or restricted issue on the repository requesting a private security contact channel without including vulnerability details in the initial public issue description.

### What to Include

To help us triage and resolve the issue quickly, please provide:
1. **Description**: A clear summary of the vulnerability and its potential impact.
2. **Component Affected**: Specify whether the issue affects the FiberForge CLI/MCP engine itself or the generated application code (e.g., specific framework templates).
3. **Reproduction Steps**: A minimal reproducible example, such as a sample `fiberforge.yaml` schema or CLI command sequence.
4. **Proof of Concept**: Any PoC scripts, payloads, or logs demonstrating the issue.
5. **Remediation Suggestions**: Any proposed fixes or mitigations, if known.

### Response Timeline

- **Acknowledgment**: We aim to acknowledge receipt of security reports within 48–72 hours.
- **Triage & Fix**: We will keep the reporter informed as we triage, develop, and test a fix.
- **Disclosure**: Coordinated public disclosure will take place alongside a release containing the fix.
