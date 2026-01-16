# CAPK Parity Gap Analysis

This document captures the remaining gaps between CAPK and CAPV paths in the Kairos CAPI providers,
and the expectations for parity with CAPI v1beta2 contracts.

## Current CAPK Gaps
1) **Control plane IP discovery**
   - Control plane kubeconfig retrieval is VSphere-only.
   - CAPK requires reading IPs from `KubevirtMachine.status.addresses`.

2) **Bootstrap secret regeneration**
   - KairosConfig regeneration watches only `VSphereMachine`.
   - CAPK needs the same behavior for `KubevirtMachine` when `spec.providerID` becomes available.

3) **ProviderID fallback in control plane**
   - Node providerID patching relies on `Machine.spec.providerID`.
   - CAPK should fall back to infra providerID when Machine spec is not yet set.

4) **Scaling/rollout behavior**
   - Control plane reconciliation is MVP-only (single-node delete).
   - Parity needs safe scale-down and basic rolling upgrade behavior.

5) **CAPI conditions**
   - Scaling conditions are not set.
   - Status fields must remain aligned with CAPI v1beta2 contract expectations.

## CAPK Expectations (latest release)
- `KubevirtMachine.spec.providerID` is authoritative for providerID.
- `KubevirtMachine.status.addresses` provides node IPs for kubeconfig retrieval.
- Bootstrap success marker must be written to `/run/cluster-api/bootstrap-success.complete`.

## Required Parity Outcomes
- ProviderID propagation and NodeRef matching work for CAPK the same as CAPV.
- Control plane kubeconfig retrieval works for CAPK using KubeVirt IPs.
- Scaling and upgrades behave consistently across CAPV and CAPK.
- CAPI v1beta2 status fields and conditions remain compliant.
