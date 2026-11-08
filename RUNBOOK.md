# GitWebhookProxy runbook

Operational procedures for GitWebhookProxy deployed with the Helm chart. Flags
and request handling are described in [README.md](README.md); chart values in
the [chart README](deployments/kubernetes/chart/gitwebhookproxy/README.md).

Examples assume release `gwp` in namespace `webhooks`:

```sh
NS=webhooks
REL=gwp
DEPLOY=gwp-gitwebhookproxy   # <release>-gitwebhookproxy unless fullnameOverride is set
```

## Service summary

| Item     | Value                                                                             |
| -------- | --------------------------------------------------------------------------------- |
| Purpose  | Validate GitHub/GitLab webhooks and forward them to an upstream (usually Jenkins) |
| Inbound  | `POST /<path>` from GitHub/GitLab, `GET /health` from probes                      |
| Outbound | HTTP(S) to `config.upstreamURL` only                                              |
| State    | None. Pods can be deleted or rescheduled at any time.                             |
| Port     | Container 8080, Service 80                                                        |
| Runs as  | uid/gid 65532, read-only root filesystem, no capabilities, no shell               |
| Secrets  | Webhook secret in a Kubernetes Secret (`secret.existingSecret`, key `secret`)     |

Impact of an outage: webhook deliveries fail and CI jobs are not triggered.
GitHub and GitLab do not retry automatically; failed deliveries can be
redelivered from the webhook settings page once the proxy is back.

## Health checks

```sh
kubectl -n $NS get deploy,pods,svc -l app.kubernetes.io/instance=$REL
kubectl -n $NS logs deploy/$DEPLOY --tail=50
helm -n $NS test $REL
```

Expected startup log:

```text
Git WebHook Proxy 0.3.0 started with provider 'github'
FIPS 140-3 mode enabled (Go Cryptographic Module v1.0.0)
Listening at: :8080
```

From inside the cluster:

```sh
kubectl -n $NS run curl --rm -it --restart=Never --image=curlimages/curl:8.11.1@sha256:c1fe1679c34d9784c1b0d1e5f62ac0a79fca01fb6377cdd33e90473c6f9f9a69 \
  --overrides='{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}},"containers":[{"name":"curl","image":"curlimages/curl:8.11.1@sha256:c1fe1679c34d9784c1b0d1e5f62ac0a79fca01fb6377cdd33e90473c6f9f9a69","args":["-s","http://'$DEPLOY'/health"],"securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}}]}}'
```

The last delivery status per webhook is visible on the provider side:
GitHub repository Settings > Webhooks > Recent Deliveries; GitLab Settings >
Webhooks > Edit > Recent events.

## Responses and log lines

| Response to sender                            | Log line                                                  | Meaning                                                                                           | Action                                                                                          |
| --------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `200` upstream body                           | `Redirected incoming request ... with Response: '200 OK'` | Forwarded                                                                                         | None                                                                                            |
| `200 Ignoring request for user: X`            | `Ignoring request for user: "X"`                          | Sender is in `ignoredUsers`, not in `allowedUser`, or empty (GitHub event without sender)         | Expected for filtered users. If unexpected, check `config.ignoredUsers` / `config.allowedUsers` |
| `400 Error validating Hook`                   | `Error Validating Hook`                                   | Signature/token does not match, or missing                                                        | See [Validation failures](#validation-failures)                                                 |
| `400 Error parsing Hook: Required header ...` | `Error Parsing Hook: ...`                                 | Provider headers missing. Usually the wrong `config.provider`, or a request that is not a webhook | Check provider setting and sender                                                               |
| `403 Not allowed to proxy path`               | `Not allowed to proxy path: "..."`                        | Path not in `allowedPaths`                                                                        | Fix the webhook URL or `config.allowedPaths`                                                    |
| `405`                                         | none                                                      | Not a `POST`                                                                                      | Webhook must use `POST`                                                                         |
| `502 Error forwarding request to upstream`    | `Error Redirecting ... to upstream ...: <error>`          | Upstream unreachable, DNS failure, TLS failure                                                    | See [Upstream failures](#upstream-failures)                                                     |
| `4xx/5xx Upstream returned status: ...`       | `Error Redirecting ..., Upstream Redirect Status: ...`    | Upstream answered with an error                                                                   | Check upstream logs (for Jenkins: the plugin endpoint and its own token/secret settings)        |

## Validation failures

1. Confirm the provider: `kubectl -n $NS get cm $DEPLOY -o jsonpath='{.data.GWP_PROVIDER}'`.
2. Confirm the secret the pod uses matches the one in GitHub/GitLab. The pod
   reads it at startup, so a changed Secret needs a restart:
   `kubectl -n $NS rollout restart deploy/$DEPLOY` (chart-managed secrets
   restart automatically through the checksum annotation; `existingSecret`
   does not).
3. GitHub: the webhook content type must be `application/json`. The signature
   is computed over the raw body; anything that rewrites the body between
   GitHub and the proxy (some WAFs, body-transforming ingress rules) breaks it.
4. GitLab: the proxy compares the `X-Gitlab-Token` header with the secret
   after trimming whitespace.
5. Reproduce with a signed request (GitHub):

   ```sh
   body='{"sender":{"login":"octocat"}}'
   sig=$(printf %s "$body" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $NF}')
   kubectl -n $NS port-forward svc/$DEPLOY 8080:80 &
   curl -i -X POST http://127.0.0.1:8080/github-webhook/ \
     -H 'Content-Type: application/json' -H 'X-GitHub-Event: push' \
     -H 'X-GitHub-Delivery: test' -H "X-Hub-Signature-256: sha256=$sig" \
     --data "$body"
   ```

## Upstream failures

- `x509: certificate signed by unknown authority`: the upstream uses a private
  CA or a self-signed certificate. Preferred fix: serve a certificate from a
  public CA, or call the upstream over the cluster-internal HTTP Service. Last
  resort: `config.insecureSkipVerify=true`, which the proxy logs as a warning
  at startup.
- TLS handshake failures with FIPS mode: the upstream must offer TLS 1.2+ with
  an approved cipher suite and key exchange (for example ECDHE with AES-GCM).
  Check with `openssl s_client -connect <host>:443 -tls1_2`.
- `connection refused`, `no such host`: check the Service name in
  `config.upstreamURL`, that the upstream pods are ready, and, if
  `networkPolicy.egress` is set, that DNS and the upstream are allowed.
- Requests time out after 30 s towards the upstream. Slow upstreams should
  acknowledge webhooks quickly and do the work asynchronously.

## Deploy and upgrade

```sh
helm -n $NS upgrade --install $REL oci://ghcr.io/schwichtgit/charts/gitwebhookproxy \
  --version <X.Y.Z> -f values.yaml --wait
helm -n $NS test $REL
```

- Verify the release artifacts first (see [Supply chain](#supply-chain)).
- Read [CHANGELOG.md](CHANGELOG.md) for upgrade notes. Upgrading from chart
  0.2.x to 0.3.0 requires a reinstall because the selector labels changed:

  ```sh
  helm -n $NS uninstall $REL
  helm -n $NS install $REL oci://ghcr.io/schwichtgit/charts/gitwebhookproxy --version 0.3.0 -f values.yaml --wait
  ```

  Webhooks fail during the gap; redeliver them afterwards. Alternatively
  install 0.3.0 under a new release name, switch the Ingress or DNS, then
  uninstall the old release.

- Keep at least two replicas and `podDisruptionBudget.enabled=true` if node
  drains must not drop webhooks.

## Rollback

```sh
helm -n $NS history $REL
helm -n $NS rollback $REL <revision> --wait
```

Rolling back across the 0.2.x/0.3.0 boundary is not possible with
`helm rollback`; reinstall the old chart instead.

## Rotate the webhook secret

GitHub and GitLab accept one secret per webhook, so rotation causes a short
window of failed deliveries.

1. Generate a secret: `openssl rand -hex 32`.
2. Update the Kubernetes Secret:

   ```sh
   kubectl -n $NS create secret generic gitwebhookproxy-secret \
     --from-literal=secret="$NEW" --dry-run=client -o yaml | kubectl apply -f -
   kubectl -n $NS rollout restart deploy/$DEPLOY
   kubectl -n $NS rollout status deploy/$DEPLOY
   ```

3. Immediately set the same secret in the GitHub/GitLab webhook settings.
4. Redeliver any deliveries that failed in between.

## Scaling

The proxy is stateless and light (requests 50m CPU / 32Mi memory by default).
Scale with `replicaCount` or enable `autoscaling` (HPA on CPU). Run at least
two replicas with the PDB enabled for node maintenance.

## Debugging without a shell

The image is distroless and has no shell. Use an ephemeral debug container
that satisfies the restricted Pod Security profile:

```sh
kubectl -n $NS debug -it pod/<pod> --image=busybox:1.37.0@sha256:a7dbc319b1438115a207e11930dc4bb3e643dcb3ec0bdd694bb36cd8fad289ec --target=gitwebhookproxy --profile=restricted
```

## Supply chain

Release images, archives and the chart are signed keyless with cosign by
`.github/workflows/release.yml` on the release tag; image SBOMs are signed
SPDX attestations. Verification commands: [README.md](README.md#verify-a-release).

With `kyverno.enabled=true` the cluster checks the signature (and SBOM) at
admission:

- Policy reports: `kubectl -n $NS get policyreport`
- Admission denials (Enforce): `kubectl -n $NS get events --field-selector reason=PolicyViolation`
- A Pod denied with `no signatures found` usually means the image was not
  built by the release workflow (for example a locally built or retagged
  image), or Kyverno is older than 1.18 and cannot read cosign v3 bundles.
- If only the SBOM check fails, set `kyverno.verifySBOM=false` and verify the
  SBOM with `cosign verify-attestation` until the Kyverno version is fixed.
- Emergency bypass: `kyverno.failureAction=Audit` (keeps reporting) rather
  than disabling the policy.

## Vulnerability response

1. Check whether the service is affected:
   `govulncheck ./...` on the release tag, and
   `trivy image ghcr.io/schwichtgit/gitwebhookproxy:<X.Y.Z>`.
2. Dependabot opens PRs for Go modules, the base images and GitHub Actions;
   security updates are opened automatically for advisories.
3. Merge the fix, cut a patch release (below), deploy, and verify the new
   image with cosign.

## Cut a release

1. Make sure `master` is green and `CHANGELOG.md` has the new version.
2. Tag and push: `git tag -s vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
3. The release workflow builds binaries, the multi-arch image
   (linux/amd64, linux/arm64) and the chart, signs them, attests the SBOMs and
   verifies the signatures before finishing.
4. Check the GitHub release page and run the verification commands from the
   README against the new version.
