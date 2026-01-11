#!/bin/bash
# Post-install hook to inject CA bundle into webhook configurations
# This script should be run after deploying the provider to ensure webhooks work correctly

set -euo pipefail

NAMESPACE="${NAMESPACE:-kairos-capi-system}"
CERT_NAME="kairos-capi-webhook-server-cert"
SECRET_NAME="kairos-capi-webhook-server-cert"
MUTATING_WEBHOOK="mutating-webhook-configuration"
VALIDATING_WEBHOOK="validating-webhook-configuration"

echo "Waiting for certificate to be ready..."
kubectl wait --for=condition=ready "certificate/${CERT_NAME}" \
  -n "${NAMESPACE}" \
  --timeout=300s || {
  echo "ERROR: Certificate not ready after 5 minutes"
  exit 1
}

echo "Certificate is ready. Extracting CA bundle..."
CA_BUNDLE=$(kubectl get secret "${SECRET_NAME}" -n "${NAMESPACE}" -o jsonpath='{.data.ca\.crt}')

if [ -z "${CA_BUNDLE}" ]; then
  echo "ERROR: Failed to extract CA bundle from secret"
  exit 1
fi

echo "Patching mutating webhook configuration..."
kubectl patch mutatingwebhookconfiguration "${MUTATING_WEBHOOK}" \
  --type='json' \
  -p="[
    {\"op\": \"replace\", \"path\": \"/webhooks/0/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"},
    {\"op\": \"replace\", \"path\": \"/webhooks/1/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}
  ]" || {
  echo "ERROR: Failed to patch mutating webhook configuration"
  exit 1
}

echo "Patching validating webhook configuration..."
kubectl patch validatingwebhookconfiguration "${VALIDATING_WEBHOOK}" \
  --type='json' \
  -p="[
    {\"op\": \"replace\", \"path\": \"/webhooks/0/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"},
    {\"op\": \"replace\", \"path\": \"/webhooks/1/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}
  ]" || {
  echo "ERROR: Failed to patch validating webhook configuration"
  exit 1
}

echo "Successfully injected CA bundle into webhook configurations"
echo "Restarting controller to pick up changes..."
kubectl rollout restart deployment/kairos-capi-controller-manager -n "${NAMESPACE}" || true

echo "Done!"
