#!/usr/bin/env bash
# K3s 1.36.1 — Single-node installation script for xyn-pos-v1
# Target: home server / cloud VM running Ubuntu 22.04+
#
# Usage:
#   chmod +x scripts/k3s/install.sh
#   sudo ./scripts/k3s/install.sh
#
# After running this script:
#   1. Run scripts/k3s/post-install.sh to set up namespaces + ArgoCD
#   2. Copy kubeconfig: sudo cat /etc/rancher/k3s/k3s.yaml > ~/.kube/config

set -euo pipefail

K3S_VERSION="v1.36.1+k3s1"
ARGOCD_VERSION="v3.4.3"

echo "==> Installing K3s ${K3S_VERSION}..."

# K3s with:
#   --disable traefik     → we install Traefik via Helm for full control
#   --disable servicelb   → we use MetalLB or cloud LB
#   --write-kubeconfig-mode 644 → allow non-root kubectl
curl -sfL https://get.k3s.io | \
  INSTALL_K3S_VERSION="${K3S_VERSION}" \
  sh -s - \
  --disable traefik \
  --disable servicelb \
  --write-kubeconfig-mode 644 \
  --kube-apiserver-arg "enable-admission-plugins=NodeRestriction" \
  --kube-apiserver-arg "audit-log-path=/var/log/k3s-audit.log"

echo "==> Waiting for K3s to be ready..."
until kubectl get nodes 2>/dev/null | grep -q "Ready"; do
  sleep 3
done
echo "    ✓ K3s node ready"

# Copy kubeconfig for current user
mkdir -p "$HOME/.kube"
sudo cp /etc/rancher/k3s/k3s.yaml "$HOME/.kube/config"
sudo chown "$(id -u):$(id -g)" "$HOME/.kube/config"
echo "    ✓ kubeconfig copied to ~/.kube/config"

echo ""
echo "==> K3s ${K3S_VERSION} installed successfully!"
echo "    Run: kubectl get nodes"
echo "    Then: ./scripts/k3s/post-install.sh"
