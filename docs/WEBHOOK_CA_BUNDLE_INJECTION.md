# Webhook CA Bundle Injection

## Overview

The Kairos CAPI Provider uses webhooks for admission control (mutation and validation). These webhooks require TLS certificates, and the Kubernetes API server needs the CA bundle to verify the webhook server's certificate.

## The Challenge

When deploying the provider, there's a chicken-and-egg problem:
1. The webhook configurations need a CA bundle to be trusted by the API server
2. The CA bundle comes from a certificate secret created by cert-manager
3. The certificate secret doesn't exist until cert-manager creates it after deployment

## Production-Ready Solution

We've implemented a **post-install Job** that automatically injects the CA bundle into webhook configurations after the certificate is ready. This is included in the deployment manifests and runs automatically.

### How It Works

1. **Certificate Creation**: cert-manager creates the certificate and secret (`kairos-capi-webhook-server-cert`)
2. **Post-Install Job**: A Kubernetes Job (`kairos-capi-webhook-ca-injection`) waits for the certificate to be ready
3. **CA Bundle Injection**: The job extracts the CA bundle from the secret and patches both webhook configurations
4. **Controller Restart**: The job restarts the controller to ensure it picks up any changes

### Files Involved

- `config/webhook/post-install-job.yaml` - The post-install job that injects the CA bundle
- `config/webhook/kustomization.yaml` - Includes the job in the webhook resources
- `config/rbac/role.yaml` - Grants permissions for patching webhooks and creating jobs
- `hack/post-install-webhook-ca-injection.sh` - Standalone script for manual injection if needed

### RBAC Permissions

The ServiceAccount (`kairos-capi-manager`) has been granted the following additional permissions:
- `mutatingwebhookconfigurations` and `validatingwebhookconfigurations`: get, list, patch, update
- `jobs`: create, get, list, watch

### Deployment Flow

```bash
# 1. Deploy the provider
kubectl apply -f components.yaml

# 2. The post-install job runs automatically
# It waits for the certificate, then patches the webhooks

# 3. Verify the job completed successfully
kubectl get job kairos-capi-webhook-ca-injection -n kairos-capi-system

# 4. Verify webhooks have CA bundles
kubectl get mutatingwebhookconfiguration mutating-webhook-configuration -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d | head -1
```

### Manual Injection (Fallback)

If the post-install job fails or you need to manually inject the CA bundle, use the provided script:

```bash
./hack/post-install-webhook-ca-injection.sh
```

Or manually:

```bash
# Wait for certificate
kubectl wait --for=condition=ready certificate/kairos-capi-webhook-server-cert \
  -n kairos-capi-system --timeout=300s

# Extract CA bundle
CA_BUNDLE=$(kubectl get secret kairos-capi-webhook-server-cert \
  -n kairos-capi-system -o jsonpath='{.data.ca\.crt}')

# Patch webhooks
kubectl patch mutatingwebhookconfiguration mutating-webhook-configuration \
  --type='json' \
  -p="[{\"op\": \"replace\", \"path\": \"/webhooks/0/clientConfig/caBundle\", \"value\": \"$CA_BUNDLE\"}, {\"op\": \"replace\", \"path\": \"/webhooks/1/clientConfig/caBundle\", \"value\": \"$CA_BUNDLE\"}]"

kubectl patch validatingwebhookconfiguration validating-webhook-configuration \
  --type='json' \
  -p="[{\"op\": \"replace\", \"path\": \"/webhooks/0/clientConfig/caBundle\", \"value\": \"$CA_BUNDLE\"}, {\"op\": \"replace\", \"path\": \"/webhooks/1/clientConfig/caBundle\", \"value\": \"$CA_BUNDLE\"}]"
```

### Certificate Rotation

When cert-manager rotates the certificate (every 90 days by default), the CA bundle may change. The post-install job only runs once, so you may need to:

1. **Option 1**: Re-run the manual injection script after certificate rotation
2. **Option 2**: Delete the job to trigger a re-run:
   ```bash
   kubectl delete job kairos-capi-webhook-ca-injection -n kairos-capi-system
   ```
3. **Option 3**: Use cert-manager's webhook injection feature (if available in your cert-manager version)

### Troubleshooting

**Webhook errors persist after deployment:**
1. Check if the certificate is ready: `kubectl get certificate -n kairos-capi-system`
2. Check if the job completed: `kubectl get job -n kairos-capi-system`
3. Check job logs: `kubectl logs job/kairos-capi-webhook-ca-injection -n kairos-capi-system`
4. Verify CA bundle is set: `kubectl get mutatingwebhookconfiguration mutating-webhook-configuration -o yaml | grep caBundle`

**Certificate not ready:**
- Check cert-manager is installed: `kubectl get pods -n cert-manager`
- Check certificate status: `kubectl describe certificate kairos-capi-webhook-server-cert -n kairos-capi-system`

**Job fails:**
- Check RBAC permissions: `kubectl auth can-i patch mutatingwebhookconfigurations --as=system:serviceaccount:kairos-capi-system:kairos-capi-manager`
- Check job logs for specific errors

## Future Improvements

1. **Cert-Manager Webhook Injection**: Use cert-manager's native webhook injection feature (requires cert-manager v1.5+ with webhook controller)
2. **Init Container**: Use an init container in the controller deployment instead of a separate job
3. **Operator Pattern**: Create a small operator that watches the certificate and automatically updates webhooks

## References

- [Kubernetes Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [CAPI Provider Development Guide](https://cluster-api.sigs.k8s.io/developer/providers/overview.html)
