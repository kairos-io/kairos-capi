#!/bin/bash
# Script to create sample cluster manifest for KubeVirt

cat > config/samples/capk/kairos_cluster_k0s_single_node.yaml <<'EOF'
# ============================================================================
# CAPK Sample: Single-Node k0s Cluster on Kairos OS with KubeVirt
# ============================================================================
#
# SETUP INSTRUCTIONS:
# 
# 1. Make sure you have completed the setup:
#    - make -f Makefile.kubevirt setup
#    - make -f Makefile.kubevirt upload-kairos-image (to upload the Kairos image)
#
# 2. Update the KubeVirtMachineTemplate below:
#    - Set the storageClassName if needed
#    - Adjust CPU, memory, and disk sizes
#    - Set the dataVolumeTemplate name to match your uploaded image PVC
#
# 3. Apply the manifest:
#    - kubectl apply -f config/samples/capk/kairos_cluster_k0s_single_node.yaml
#
# ============================================================================

apiVersion: cluster.x-k8s.io/v1beta2
kind: Cluster
metadata:
  name: kairos-cluster
  namespace: default
spec:
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: KubevirtCluster
    name: kairos-cluster
  controlPlaneRef:
    apiGroup: controlplane.cluster.x-k8s.io
    kind: KairosControlPlane
    name: kairos-control-plane
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: KubevirtCluster
metadata:
  name: kairos-cluster
  namespace: default
spec: {}
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta2
kind: KairosControlPlane
metadata:
  name: kairos-control-plane
  namespace: default
spec:
  replicas: 1
  version: "v1.34.1+k0s.1"
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
      kind: KubevirtMachineTemplate
      name: kairos-control-plane-template
      namespace: default
  kairosConfigTemplate:
    name: kairos-config-template-control-plane
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: KubevirtMachineTemplate
metadata:
  name: kairos-control-plane-template
  namespace: default
spec:
  template:
    spec:
      virtualMachineTemplate:
        spec:
          dataVolumeTemplates:
          - apiVersion: cdi.kubevirt.io/v1beta1
            kind: DataVolume
            metadata:
              name: kairos-rootdisk
            spec:
              pvc:
                accessModes:
                - ReadWriteOnce
                resources:
                  requests:
                    storage: 20Gi
                # TODO: Set storageClassName if needed
                # storageClassName: standard
              source:
                pvc:
                  name: kairos-kubevirt
                  namespace: default
          running: true
          template:
            spec:
              domain:
                cpu:
                  cores: 2
                memory:
                  guest: 4Gi
                devices:
                  disks:
                  - name: rootdisk
                    disk:
                      bus: virtio
                  interfaces:
                  - name: default
                    masquerade: {}
              networks:
              - name: default
                pod: {}
              volumes:
              - name: rootdisk
                dataVolume:
                  name: kairos-rootdisk
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta2
kind: KairosConfigTemplate
metadata:
  name: kairos-config-template-control-plane
  namespace: default
spec:
  template:
    spec:
      role: control-plane
      distribution: k0s
      kubernetesVersion: "v1.34.1+k0s.1"
      userName: kairos
      userPassword: kairos
      userGroups:
        - admin
      # Optional: Add GitHub user for SSH access
      # githubUser: "your-github-username"
      # Optional: Add SSH public key instead
      # sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E..."
EOF

echo "Sample cluster manifest created at config/samples/capk/kairos_cluster_k0s_single_node.yaml"
echo "Remember to update the dataVolumeTemplate source.pvc.name if your PVC name differs from 'kairos-kubevirt'"
