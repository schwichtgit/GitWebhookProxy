# gitwebhookproxy Helm chart

Deploys GitWebhookProxy, a proxy that validates GitHub and GitLab webhook
signatures or tokens and forwards valid requests to an upstream URL.

Requirements: Kubernetes 1.27 or later, Helm 3.8 or later.

## Install

Create the webhook secret first, then install the chart:

```sh
kubectl create secret generic gitwebhookproxy-secret \
  --from-literal=secret='<webhook secret>'

helm install gwp ./deployments/kubernetes/chart/gitwebhookproxy \
  --set config.upstreamURL=http://jenkins.jenkins.svc:8080 \
  --set secret.existingSecret=gitwebhookproxy-secret
```

`config.upstreamURL` is required. Rendering fails with an error if it is empty.

If neither `secret.existingSecret` nor `secret.value` is set, the proxy starts
without a secret and forwards requests without validating them. The release
notes print a warning in that case.

`secret.value` creates a Secret from a chart value. The value is then stored in
the Helm release history, so `secret.existingSecret` is the better choice
outside of test setups.

Run the connection test after installing:

```sh
helm test gwp
```

The test pod runs `busybox` and requests `/health` through the Service. If
`networkPolicy.ingressFrom` is restricted, the test pod (label
`app.kubernetes.io/component: test`) must be allowed as a source, or the test
fails.

## Configuration

The proxy reads its settings from `GWP_*` environment variables. The chart
writes the `config` values into a ConfigMap that is loaded with `envFrom`, and
injects the webhook secret as `GWP_SECRET` from a Secret. Changing the
ConfigMap or the chart-managed Secret restarts the pods through checksum
annotations. Changes to an existing Secret referenced with
`secret.existingSecret` need a manual restart
(`kubectl rollout restart deployment/<name>`).

| Value | Environment variable | Default |
|-------|----------------------|---------|
| `config.provider` | `GWP_PROVIDER` | `github` |
| `config.upstreamURL` | `GWP_UPSTREAMURL` | `""` (required) |
| `config.allowedPaths` | `GWP_ALLOWEDPATHS` | `""` |
| `config.ignoredUsers` | `GWP_IGNOREDUSERS` | `""` |
| `config.allowedUsers` | `GWP_ALLOWEDUSER` | `""` |
| `config.insecureSkipVerify` | `GWP_INSECURESKIPVERIFY` | `false` |
| `secret.value` or `secret.existingSecret` | `GWP_SECRET` | not set |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `replicaCount` | int | `1` | Number of replicas. Ignored when `autoscaling.enabled` is true. |
| `image.repository` | string | `ghcr.io/schwichtgit/gitwebhookproxy` | Image repository. |
| `image.tag` | string | `""` | Image tag. Defaults to the chart `appVersion`. |
| `image.digest` | string | `""` | Image digest (`sha256:...`). Used instead of the tag when set. |
| `image.pullPolicy` | string | `IfNotPresent` | Image pull policy. |
| `imagePullSecrets` | list | `[]` | Pull secrets for the image. |
| `nameOverride` | string | `""` | Replaces the chart name in resource names. |
| `fullnameOverride` | string | `""` | Replaces the full resource name. |
| `config.provider` | string | `github` | Git provider: `github` or `gitlab`. |
| `config.upstreamURL` | string | `""` | URL that validated requests are forwarded to. Required. |
| `config.allowedPaths` | string | `""` | Comma-separated allowed request paths. Empty allows all. |
| `config.ignoredUsers` | string | `""` | Comma-separated users whose events are dropped. |
| `config.allowedUsers` | string | `""` | Comma-separated users whose events are forwarded. Empty allows all. |
| `config.insecureSkipVerify` | bool | `false` | Skip TLS verification of the upstream. Only for trusted upstreams with self-signed certificates. |
| `secret.value` | string | `""` | Webhook secret. Creates a Secret managed by the chart. |
| `secret.existingSecret` | string | `""` | Name of an existing Secret with the webhook secret. Mutually exclusive with `secret.value`. |
| `secret.existingSecretKey` | string | `secret` | Key in the existing Secret. |
| `extraEnv` | list | `[]` | Additional container environment variables, for example `GODEBUG=fips140=on`. |
| `serviceAccount.create` | bool | `true` | Create a ServiceAccount. |
| `serviceAccount.name` | string | `""` | ServiceAccount name. Generated from the full name if empty. |
| `serviceAccount.annotations` | object | `{}` | ServiceAccount annotations. |
| `serviceAccount.automount` | bool | `false` | Mount the API token. The proxy does not use the Kubernetes API. |
| `podAnnotations` | object | `{}` | Extra pod annotations. |
| `podLabels` | object | `{}` | Extra pod labels. |
| `podSecurityContext` | object | see `values.yaml` | Pod security context. Non-root, UID/GID 65532, `RuntimeDefault` seccomp. |
| `securityContext` | object | see `values.yaml` | Container security context. Read-only root filesystem, no privilege escalation, all capabilities dropped. |
| `service.type` | string | `ClusterIP` | Service type. |
| `service.port` | int | `80` | Service port. The container listens on 8080. |
| `service.annotations` | object | `{}` | Service annotations. |
| `ingress.enabled` | bool | `false` | Create an Ingress. |
| `ingress.className` | string | `""` | IngressClass name. |
| `ingress.annotations` | object | `{}` | Ingress annotations. |
| `ingress.hosts` | list | see `values.yaml` | Hosts and paths. |
| `ingress.tls` | list | `[]` | TLS configuration. |
| `resources` | object | requests 50m CPU / 32Mi, limit 128Mi | Container resources. No CPU limit by default. |
| `livenessProbe` | object | HTTP GET `/health` | Liveness probe. |
| `readinessProbe` | object | HTTP GET `/health` | Readiness probe. |
| `autoscaling.enabled` | bool | `false` | Create a HorizontalPodAutoscaler (`autoscaling/v2`). |
| `autoscaling.minReplicas` | int | `2` | Minimum replicas. |
| `autoscaling.maxReplicas` | int | `5` | Maximum replicas. |
| `autoscaling.targetCPUUtilizationPercentage` | int | `80` | CPU target. |
| `autoscaling.targetMemoryUtilizationPercentage` | int | not set | Memory target. |
| `podDisruptionBudget.enabled` | bool | `false` | Create a PodDisruptionBudget. |
| `podDisruptionBudget.minAvailable` | int or string | `1` | Used unless `maxUnavailable` is set. With one replica this blocks node drains. |
| `podDisruptionBudget.maxUnavailable` | int or string | `""` | Takes precedence over `minAvailable` when set. |
| `networkPolicy.enabled` | bool | `false` | Create a NetworkPolicy. |
| `networkPolicy.ingressFrom` | list | `[]` | Allowed sources for the proxy port. Empty allows all sources. |
| `networkPolicy.egress` | list | `[]` | Egress rules. Empty leaves egress unrestricted. If set, allow DNS and the upstream. |
| `nodeSelector` | object | `{}` | Node selector. |
| `tolerations` | list | `[]` | Tolerations. |
| `affinity` | object | `{}` | Affinity rules. |
| `topologySpreadConstraints` | list | `[]` | Topology spread constraints. Values are passed through `tpl`. |
| `priorityClassName` | string | `""` | PriorityClass name. |
| `gke.backendConfig.enabled` | bool | `false` | Create a GKE BackendConfig and reference it from the Service. |
| `gke.backendConfig.name` | string | `""` | BackendConfig name. Defaults to `<fullname>-backend`. |
| `gke.backendConfig.healthCheck.*` | object | `/health` on port 8080 | Load balancer health check. Set `enabled: false` to omit it. |
| `gke.backendConfig.extraSpec` | object | `{}` | Additional BackendConfig spec fields, such as `timeoutSec`, `logging` or `securityPolicy`. |
| `gke.frontendConfig.enabled` | bool | `false` | Create a GKE FrontendConfig and reference it from the Ingress. Requires `ingress.enabled`. |
| `gke.frontendConfig.name` | string | `""` | FrontendConfig name. Defaults to `<fullname>-frontend`. |
| `gke.frontendConfig.sslPolicy` | string | `""` | Name of an existing GCP SSL policy. |
| `gke.frontendConfig.redirectToHttps.enabled` | bool | `false` | Redirect HTTP to HTTPS at the load balancer. |
| `gke.frontendConfig.redirectToHttps.responseCodeName` | string | `MOVED_PERMANENTLY_DEFAULT` | Redirect response code. |
| `kyverno.enabled` | bool | `false` | Create a namespaced Kyverno Policy that verifies the image signature. |
| `kyverno.failureAction` | string | `Audit` | `Audit` reports violations, `Enforce` blocks the Pod. |
| `kyverno.mutateDigest` | bool | `true` | Rewrite the image reference to the verified digest. |
| `kyverno.verifySBOM` | bool | `true` | Also require the signed SPDX SBOM attestation. |
| `kyverno.webhookTimeoutSeconds` | int | `30` | Admission webhook timeout (1-30). |
| `kyverno.keyless.issuer` | string | `https://token.actions.githubusercontent.com` | OIDC issuer of the signing certificate. |
| `kyverno.keyless.subjectRegExp` | string | release workflow on `v*` tags | Certificate identity of the release workflow. |
| `kyverno.keyless.rekorURL` | string | `https://rekor.sigstore.dev` | Rekor transparency log. |

## GKE Ingress

Example for the GKE external Application Load Balancer:

```yaml
service:
  # GKE Ingress needs NodePort, or ClusterIP with container-native load
  # balancing (annotation cloud.google.com/neg: '{"ingress": true}').
  type: NodePort
ingress:
  enabled: true
  annotations:
    kubernetes.io/ingress.class: gce
    kubernetes.io/ingress.global-static-ip-name: gwp-ip
  hosts:
    - host: hooks.example.com
      paths:
        - path: /
          pathType: Prefix
gke:
  backendConfig:
    enabled: true
  frontendConfig:
    enabled: true
    sslPolicy: modern-ssl-policy
    redirectToHttps:
      enabled: true
```

The BackendConfig health check points to `/health` on the container port, so
the load balancer does not fall back to `GET /`.

## Image verification with Kyverno

Release images are signed keyless with cosign by the release workflow, and
each platform image carries a signed SPDX SBOM attestation. With
`kyverno.enabled=true` the chart creates a namespaced Kyverno `Policy` that
checks both before a Pod starts:

- the signature must come from
  `https://github.com/schwichtgit/GitWebhookProxy/.github/workflows/release.yml`
  on a `v*` tag, issued by `https://token.actions.githubusercontent.com`, and be
  recorded in Rekor
- the SPDX SBOM attestation must be signed by the same identity
- the image reference is rewritten to the verified digest (`mutateDigest`)

Requirements and notes:

- Kyverno 1.18 or later. The policy uses `verifyImages` with
  `type: SigstoreBundle`, which reads the bundle format that cosign v3 writes.
  The older `Cosign` verifier type does not see these signatures.
- Start with `failureAction: Audit`, check the policy reports
  (`kubectl get policyreport -n <namespace>`), then switch to `Enforce`.
- Some Kyverno versions have reported problems verifying cosign v3
  attestations. If the SBOM check fails while the signature check passes, set
  `kyverno.verifySBOM=false` and verify SBOMs with `cosign verify-attestation`
  instead.
- The policy only matches this chart's image repository. Cluster-wide Pod
  Security is left to Pod Security Admission or a platform-level policy set;
  the chart already meets the restricted profile, and CI checks it against
  Kyverno's baseline and restricted policies.

## Pod Security

The default values meet the Kubernetes Pod Security `restricted` profile. The
image is a distroless static image that runs as UID 65532 and writes no files.

## Upgrading from chart 0.2.x

Chart 0.3.0 is not upgrade-compatible with 0.2.x. The values layout changed and
the Deployment selector labels changed. The selector is immutable, so
`helm upgrade` of an existing 0.2.x release fails. Uninstall the old release and
install the new chart, or install under a different release name and switch
traffic.

Values mapping:

| 0.2.x | 0.3.0 |
|-------|-------|
| `gitWebhookProxy.replicas` | `replicaCount` |
| `gitWebhookProxy.image.name` | `image.repository` (now `ghcr.io/schwichtgit/gitwebhookproxy`) |
| `gitWebhookProxy.image.tag` | `image.tag` (defaults to `<appVersion>`) |
| `gitWebhookProxy.useCustomName`, `customName` | `fullnameOverride` |
| `gitWebhookProxy.config.*` | `config.*` |
| `gitWebhookProxy.config.secret` | `secret.value` |
| `gitWebhookProxy.existingSecretName` | `secret.existingSecret` (key set by `secret.existingSecretKey`, default `secret`) |
| `gitWebhookProxy.resources` | `resources` |
| `gitWebhookProxy.securityContext` | `securityContext` |
| `gitWebhookProxy.podSecurityContext` | `podSecurityContext` (UID changed from 1001 to 65532) |
| `gitWebhookProxy.service.ports.servicePort` | `service.port` |
| `gitWebhookProxy.service.annotations` | `service.annotations` |
| `gitWebhookProxy.service.labels` | removed |
| `gitWebhookProxy.labels.*` | removed (standard `app.kubernetes.io/*` labels) |
| `gitWebhookProxy.ingress.host`, `tlsSecretName` | `ingress.hosts`, `ingress.tls` |
| `gitWebhookProxy.ingress.annotations` | `ingress.annotations` |
| `gitWebhookProxy.ingress.gcp.enabled`, `name` | `gke.frontendConfig.enabled`, `name` |
| `gitWebhookProxy.ingress.gcp.sslPolicy.name` | `gke.frontendConfig.sslPolicy` |
| `gitWebhookProxy.ingress.gcp.redirects.*` | `gke.frontendConfig.redirectToHttps.*` |
| `gitWebhookProxy.ingress.gcp.backend.enabled`, `name` | `gke.backendConfig.enabled`, `name` |
| `gitWebhookProxy.ingress.gcp.backend.healthCheck.*` | `gke.backendConfig.healthCheck.*` |
| `gitWebhookProxy.ingress.gcp.backend.healthCheck.proxyHeader`, `logConfig` | removed; set additional fields through `gke.backendConfig.extraSpec` |

Other changes:

- The Service type defaults to `ClusterIP` instead of `NodePort`.
- `config.allowedUsers` is now passed as `GWP_ALLOWEDUSER`, the variable the
  proxy reads. The 0.2.x chart set `GWP_ALLOWEDUSERS`, which the proxy ignored.
- The stakater Reloader annotations were removed. Pods restart on config
  changes through checksum annotations.
- `gke.backendConfig.enabled` no longer requires `ingress.enabled` or the
  FrontendConfig to be enabled.
- No Secret is created when no webhook secret is configured.
- Probe timings are set through `livenessProbe` and `readinessProbe` and no
  longer depend on the GKE health check values.
