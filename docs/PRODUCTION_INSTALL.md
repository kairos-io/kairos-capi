# Production Installation (CAPV/CAPK)

This guide covers a production-friendly installation flow for the Kairos
bootstrap and control-plane providers, and notes what you must install
separately for CAPK.

## Prerequisites

- Management cluster (Kubernetes 1.28+).
- A reachable container registry for controller images.
- For CAPK: KubeVirt + CDI installed and a default StorageClass available.
- `kubectl`, `kustomize`, and `clusterctl` available in your environment.

## Kairos Providers (Bootstrap + ControlPlane)

1) Build and push the controller image:

```bash
export IMG=registry.example.com/kairos-capi:v0.1.0
make docker-build
make docker-push
```

2) Ensure the controller pulls from your registry in-cluster:

- Remove the local-only patch from `config/manager/kustomization.yaml`:

```yaml
patches:
- path: imagepullpolicy_patch.yaml
```

This patch sets `imagePullPolicy: Never` (good for kind/local, not for
production).

3) Deploy the provider components:

```bash
export IMG=registry.example.com/kairos-capi:v0.1.0
make install
make deploy
```

4) Verify:

```bash
kubectl get pods -n kairos-capi-system
kubectl get crds | grep kairos
```

## CAPK (Infrastructure Provider) and KubeVirt

CAPK is a separate infrastructure provider and must be installed in the
management cluster alongside KubeVirt/CDI.

1) Install KubeVirt and CDI (versioned to your environment).
2) Install CAPK:

```bash
clusterctl init --infrastructure kubevirt
```

3) Use CAPK-compatible samples (v1alpha4) from `config/samples/capk/`.

## Notes

- If you need custom image pull policies, create a kustomize overlay rather
  than modifying `config/manager` directly.
- `make generate` / `make manifests` are only needed when you change APIs,
  CRDs, or controller code; they are not required for routine installs.

