# Quick Start Guide - CAPD (Docker)

This guide walks you through creating a single-node k0s cluster on Kairos using Cluster API with the Docker provider (CAPD).

## Prerequisites

1. **Management Cluster**: A Kubernetes cluster (can be kind, minikube, or any Kubernetes cluster)
2. **Cluster API**: CAPI v1.11+ installed
3. **CAPD**: Cluster API Provider Docker installed
4. **Kairos CAPI Provider**: This provider installed

### Installing Prerequisites

#### 1. Install Cluster API

```bash
# Install clusterctl
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.0/clusterctl-linux-amd64 -o clusterctl
chmod +x clusterctl
sudo mv clusterctl /usr/local/bin/

# Initialize CAPI
clusterctl init --infrastructure docker
```

#### 2. Install Kairos CAPI Provider

```bash
# Apply CRDs
kubectl apply -f config/crd/bases

# Apply RBAC
kubectl apply -f config/rbac

# Deploy the manager (or build and deploy your own image)
kubectl apply -f config/manager/manager.yaml
```

## Creating a Cluster

### Step 1: Review the Sample Manifest

The sample manifest is located at `config/samples/capd/kairos_cluster_k0s_single_node.yaml`.

Key components:
- `Cluster` - References DockerCluster and KairosControlPlane
- `DockerCluster` - Infrastructure cluster for Docker
- `KairosControlPlane` - Control plane with replicas=1 (single-node)
- `DockerMachineTemplate` - Template for Docker machines
- `KairosConfigTemplate` - Bootstrap configuration template

### Step 2: Customize the Manifest (Optional)

Edit `config/samples/capd/kairos_cluster_k0s_single_node.yaml`:

- Update `spec.version` in `KairosControlPlane` to your desired k0s version
- Add `githubUser` or `sshPublicKey` in `KairosConfigTemplate` for SSH access
- Change `userName`/`userPassword` if needed (default: kairos/kairos)

### Step 3: Apply the Manifest

```bash
kubectl apply -f config/samples/capd/kairos_cluster_k0s_single_node.yaml
```

### Step 4: Wait for Cluster to be Ready

```bash
# Watch the cluster status
kubectl get cluster kairos-cluster -w

# Check control plane status
kubectl get kairoscontrolplane kairos-control-plane

# Check machines
kubectl get machines

# Once ready, check nodes
kubectl get nodes --kubeconfig=<path-to-kubeconfig>
```

### Step 5: Verify k0s is Running

```bash
# Get kubeconfig for the cluster
clusterctl get kubeconfig kairos-cluster > kairos-kubeconfig.yaml

# Check nodes
kubectl --kubeconfig=kairos-kubeconfig.yaml get nodes

# Check k0s pods (if accessible)
kubectl --kubeconfig=kairos-kubeconfig.yaml get pods -n kube-system
```

## Troubleshooting

### Cluster Not Ready

```bash
# Check cluster conditions
kubectl describe cluster kairos-cluster

# Check control plane conditions
kubectl describe kairoscontrolplane kairos-control-plane

# Check machine status
kubectl describe machine <machine-name>

# Check bootstrap config
kubectl describe kairosconfig <kairosconfig-name>
kubectl get secret <dataSecretName> -o yaml
```

### Bootstrap Data Not Generated

```bash
# Check bootstrap controller logs
kubectl logs -n kairos-capi-system deployment/kairos-capi-controller-manager | grep bootstrap

# Verify KairosConfig has required fields
kubectl get kairosconfig -o yaml
```

### Worker Token Issues

For worker nodes, ensure:
- `WorkerToken` or `WorkerTokenSecretRef` is set in `KairosConfig`
- Token secret exists and contains the correct key
- Token is valid for joining the cluster

## Next Steps

- Add worker nodes using `MachineDeployment` with `KairosConfigTemplate` for workers
- Configure additional manifests via `spec.manifests` in `KairosConfig`
- Explore multi-node control plane (set `spec.replicas > 1`)

## Cleanup

```bash
# Delete the cluster
kubectl delete -f config/samples/capd/kairos_cluster_k0s_single_node.yaml

# Or use clusterctl
clusterctl delete cluster kairos-cluster
```

