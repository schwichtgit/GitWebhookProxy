# Configure a GitHub webhook

These steps create a repository webhook that sends events through
GitWebhookProxy. Organization webhooks work the same way under the
organization settings.

1. Open the repository **Settings** tab and select **Webhooks**.

   ![Webhooks settings][webhooks]

2. Select **Add webhook**.

   ![Add webhook][add-webhook]

3. Fill in the form:
   - **Payload URL**: `https://<proxy host>/<path>`. The path is forwarded to
     the upstream unchanged and must match `allowedPaths` if that is set. For
     the Jenkins GitHub plugin, use `/github-webhook/`.
   - **Content type**: `application/json`.
   - **Secret**: the same value as the proxy secret (`GWP_SECRET`, or the
     Kubernetes Secret referenced by `secret.existingSecret`). Generate one
     with `openssl rand -hex 32`.

   ![Webhook details][webhook-details]

4. Choose the events to send. The proxy reads the sender for `push`,
   `pull_request` and `issue_comment` events. GitHub events of other types
   have no sender as far as the proxy is concerned and are ignored.
5. Keep **Active** checked and select **Add webhook**.

   ![Save webhook][save-webhook]

GitHub sends a `ping` event after saving. The proxy answers it with
`200 Ignoring request for user:` because `ping` is not forwarded; this is
expected. Send a test push and check **Recent Deliveries** for a `200` with the
upstream response.

GitHub signs each delivery with both `X-Hub-Signature-256` (HMAC-SHA256) and
`X-Hub-Signature` (HMAC-SHA1). The proxy validates the SHA-256 signature.

Troubleshooting: [RUNBOOK.md](../RUNBOOK.md#validation-failures). GitHub
documentation: <https://docs.github.com/en/webhooks>.

[webhooks]: images/webhooks.png "Webhooks"
[add-webhook]: images/add-webhook.png "Add webhook"
[webhook-details]: images/webhook-details.png "Webhook details"
[save-webhook]: images/save-webhook.png "Save webhook"
