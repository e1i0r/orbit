# Security Policy

## Supported Versions

Orbit is under active development. Security updates and bug fixes are applied to the latest release on the `main` branch.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

---

## Reporting a Vulnerability

The Orbit team takes security vulnerabilities seriously. We appreciate your efforts to responsibly disclose your findings.

### How to Report

- **Do NOT file a public issue** on GitHub for security vulnerabilities.
- Please report vulnerabilities privately via **GitHub Security Advisories** on the repository:  
  [https://github.com/e1i0r/orbit/security/advisories/new](https://github.com/e1i0r/orbit/security/advisories/new)

### What to Include in Your Report

To help us triage and resolve the issue quickly, please provide:
1. A detailed description of the vulnerability and its potential impact.
2. Step-by-step reproduction instructions or a minimal proof of concept (PoC).
3. The environment details (OS, architecture, Go version, Orbit version).
4. Any suggested mitigations or patches if available.

### Response & Disclosure Process

1. **Acknowledgment:** We will acknowledge receipt of your report within 48 hours.
2. **Investigation:** We will evaluate the vulnerability, assess its severity, and determine an appropriate fix.
3. **Resolution:** A fix will be developed in a private security branch and validated against our test suites.
4. **Coordinated Disclosure:** We will publish a patched release along with a security advisory crediting you for the discovery.
