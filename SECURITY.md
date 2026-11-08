# Security policy

## Supported versions

| Version | Supported |
| ------- | --------- |
| 0.3.x   | Yes       |
| < 0.3   | No        |

Releases before 0.3.0 were published by the upstream project
(stakater/GitWebhookProxy) and are not maintained here.

## Reporting a vulnerability

Report vulnerabilities privately through
[GitHub security advisories](https://github.com/schwichtgit/GitWebhookProxy/security/advisories/new).
Do not open a public issue.

Include the affected version, the configuration (provider, secret set or
not, deployment method), steps to reproduce, and the impact you observed.

You can expect an acknowledgement within 5 working days. Fixes are released
as a patch version and announced in the release notes and CHANGELOG.md.

## Scope

In scope:

- Bypassing webhook signature or token validation
- Bypassing path or user filtering
- Request smuggling or header injection towards the upstream
- Disclosure of the webhook secret or upstream details
- Vulnerabilities in the container image, Helm chart defaults or release
  artifacts

Out of scope:

- Running the proxy without a secret (requests are not validated by design;
  see README.md)
- Denial of service through request volume
- Findings in third-party upstream services

## Supply chain

Release binaries, container images and the Helm chart are signed keyless with
cosign by the release workflow, and images carry signed SPDX SBOM
attestations. Verification commands are in
[README.md](README.md#verify-a-release). Dependencies are monitored by
Dependabot and govulncheck; images are scanned with trivy in CI and before
each release is signed.
