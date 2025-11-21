# Kairos CAPI Provider - Design Document

## Overview

This document describes the design and implementation of the Kairos CAPI Provider, which consists of two Cluster API providers:

1. **Bootstrap Provider** - Generates Kairos cloud-config bootstrap data
2. **Control Plane Provider** - Manages Kairos-based Kubernetes control plane machines

## Architecture

### Bootstrap Provider

The Bootstrap Provider implements the BootstrapConfig contract (v1beta2) and generates Kairos-compatible cloud-config for Machines.

#### API Types

- `KairosConfig` - Bootstrap configuration for a single Machine
- `KairosConfigTemplate` - Template for creating multiple KairosConfig resources

#### Key Fields

- `spec.role`: `"control-plane"` | `"worker"`
- `spec.distribution`: `"k0s"` | `"k3s"` (MVP: k0s only)
- `spec.kubernetesVersion`: Kubernetes version to install
- `spec.serverAddress`: API server address (for worker nodes)
- `spec.token` / `spec.tokenSecretRef`: Join token for worker nodes

#### Contract Compliance

- [x] `status.dataSecretName` - Points to Secret containing bootstrap data
- [x] `status.ready` - Indicates bootstrap data is ready
- [x] `status.conditions` - Standard CAPI conditions

#### Cloud-Config Generation

The controller generates Kairos cloud-config that:

1. Installs k0s using the official installer script
2. Configures k0s based on role:
   - **Control-plane**: Creates k0s controller configuration and starts k0scontroller service
   - **Worker**: Uses join token and server address to join existing cluster
3. Supports pre/post commands and file writes via Kairos stages

### Control Plane Provider

The Control Plane Provider implements the ControlPlane contract (v1beta2) and manages control plane Machines.

#### API Types

- `KairosControlPlane` - Manages control plane Machines
- `KairosControlPlaneTemplate` - Template for creating KairosControlPlane resources

#### Key Fields

- `spec.replicas`: Number of control plane machines (MVP: 1)
- `spec.version`: Kubernetes version
- `spec.machineTemplate.infrastructureRef`: Reference to infrastructure template
- `spec.kairosConfigTemplate`: Reference to KairosConfigTemplate

#### Contract Compliance

- [x] `status.initialized` - Indicates control plane is initialized
- [x] `status.readyReplicas` - Number of ready control plane machines
- [x] `status.updatedReplicas` - Number of machines with desired version
- [x] `status.unavailableReplicas` - Number of unavailable machines
- [x] `status.conditions` - Standard CAPI conditions

#### Machine Management

The controller:

1. Creates/updates/deletes Machines to match desired replica count
2. Creates KairosConfig resources for each Machine
3. Clones infrastructure templates to create InfraMachine resources
4. Tracks Machine and Node status to update KCP status

## Implementation Status

### Phase 0: Repo Bootstrap [COMPLETED]

- [x] Go module initialization
- [x] Kubebuilder-style project structure
- [x] Manager setup with scheme registration

### Phase 1: Minimal Bootstrap Provider [COMPLETED]

- [x] API types (KairosConfig, KairosConfigTemplate)
- [x] Bootstrap controller reconcile loop
- [x] Kairos cloud-config generation for k0s
- [ ] CRD generation and RBAC manifests
- [ ] Sample YAML manifests

### Phase 2: Enhance Kairos Bootstrap Semantics [IN PROGRESS]

- [ ] Align with Kairos cloud-init documentation
- [ ] Support Kairos stages properly
- [ ] Add validation webhooks

### Phase 3: Minimal Control Plane Provider [COMPLETED]

- [x] API types (KairosControlPlane, KairosControlPlaneTemplate)
- [x] Control plane controller reconcile loop
- [x] Machine creation and management
- [ ] Infrastructure machine cloning (placeholder)
- [ ] Sample YAML manifests

### Phase 4: Testing and Polish [IN PROGRESS]

- [ ] Unit tests
- [ ] Integration tests
- [ ] Documentation

## Known Limitations (MVP)

1. **Single Control Plane**: Only supports single control plane node (no HA)
2. **Infrastructure Integration**: Infrastructure machine creation is a placeholder - requires provider-specific implementation
3. **k0s Only**: k3s distribution not yet implemented
4. **No Upgrades**: Upgrade logic is minimal
5. **API Version**: Currently using v1beta2 API types, but compatible with CAPI v1.8+ (may need adjustment for v1.11+)

## Future Enhancements

1. **HA Control Plane**: Support for multiple control plane nodes
2. **k3s Support**: Implement k3s distribution
3. **Upgrade Support**: Sophisticated rolling upgrade logic
4. **ClusterClass Support**: Full ClusterClass compatibility
5. **Infrastructure Provider Integration**: Complete integration with CAPV, CAPD, etc.

## Contract Compliance Notes

### BootstrapConfig Contract (v1beta2)

- [x] `status.dataSecretName` - Required field, implemented
- [x] `status.ready` - Required field, implemented
- [x] Conditions - Recommended, implemented
- [x] Pause support - Implemented via `spec.pause`

### ControlPlane Contract (v1beta2)

- [x] `status.initialized` - Required field, implemented
- [x] `status.readyReplicas` - Required field, implemented
- [x] `status.updatedReplicas` - Required field, implemented
- [x] `status.unavailableReplicas` - Required field, implemented
- [x] Conditions - Recommended, implemented
- [NOTE] Endpoint management - Deferred to infrastructure provider

## References

- [Cluster API BootstrapConfig Contract](https://cluster-api.sigs.k8s.io/developer/providers/contracts/bootstrap-config)
- [Cluster API ControlPlane Contract](https://cluster-api.sigs.k8s.io/developer/providers/contracts/control-plane)
- [Kairos Cloud-Init Documentation](https://kairos.io/docs/architecture/cloud-init/)
- [Kairos Configuration Reference](https://kairos.io/docs/reference/configuration/)

