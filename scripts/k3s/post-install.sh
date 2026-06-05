#!/usr/bin/env bash
# K3s post-install: namespaces → Traefik → cert-manager → Sealed Secrets → ArgoCD
# Run AFTER scripts/k3s/install.sh
#
# Usage: ./scripts/k3s/post-install.sh

set -euo pipefail

ARGOCD_VERSION="v3.4.3"
SEALED_SECRETS_VERSION="v0.27.3"
CERT_MANAGER_VERSION="v1.17.2"

step() { echo ""; echo "==> $1"; }
ok()   { echo "    ✓ $1"; }

# ─────────────────────────────────────────────────────────
# 1. Namespaces
# ─────────────────────────────────────────────────────────
step "Creating namespaces..."
kubectl apply -f infra/k8s/base/namespaces/namespaces.yml
ok "Namespaces created"

# ─────────────────────────────────────────────────────────
# 2. Helm setup
# ─────────────────────────────────────────────────────────
step "Adding Helm repos..."
helm repo add traefik         https://traefik.github.io/charts
helm repo add jetstack        https://charts.jetstack.io
helm repo add sealed-secrets  https://bitnami-labs.github.io/sealed-secrets
helm repo add argo             https://argoproj.github.io/argo-helm
helm repo update
ok "Helm repos updated"

# ─────────────────────────────────────────────────────────
# 3. Traefik ingress controller
# ─────────────────────────────────────────────────────────
step "Installing Traefik..."
helm upgrade --install traefik traefik/traefik \
  --namespace kube-system \
  --values infra/k8s/base/ingress/traefik-values.yml \
  --wait
ok "Traefik installed"

# ─────────────────────────────────────────────────────────
# 4. cert-manager (Let's Encrypt TLS)
# ─────────────────────────────────────────────────────────
step "Installing cert-manager ${CERT_MANAGER_VERSION}..."
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version "${CERT_MANAGER_VERSION}" \
  --set installCRDs=true \
  --wait
kubectl apply -f infra/k8s/base/ingress/cluster-issuer.yml
ok "cert-manager installed + ClusterIssuer created"

# ─────────────────────────────────────────────────────────
# 5. Sealed Secrets controller
# ─────────────────────────────────────────────────────────
step "Installing Sealed Secrets ${SEALED_SECRETS_VERSION}..."
helm upgrade --install sealed-secrets sealed-secrets/sealed-secrets \
  --namespace kube-system \
  --version "${SEALED_SECRETS_VERSION}" \
  --set fullnameOverride=sealed-secrets-controller \
  --wait

# Backup the master key immediately after install
echo ""
echo "  ⚠️  Backing up Sealed Secrets master key..."
kubectl get secret \
  -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  -o yaml > infra/k8s/base/sealed-secrets/master-key-backup.yaml
echo "  ✓ Master key saved to infra/k8s/base/sealed-secrets/master-key-backup.yaml"
echo "  ⚠️  STORE THIS FILE SECURELY — without it you cannot decrypt sealed secrets!"
echo "  ⚠️  Add it to your password manager or vault. DO NOT commit it to git."

# ─────────────────────────────────────────────────────────
# 6. Persistent volumes (local path provisioner)
# ─────────────────────────────────────────────────────────
step "Applying storage classes..."
kubectl apply -f infra/k8s/base/storage/storage-classes.yml
ok "Storage classes applied"

# ─────────────────────────────────────────────────────────
# 7. ArgoCD
# ─────────────────────────────────────────────────────────
step "Installing ArgoCD ${ARGOCD_VERSION}..."
helm upgrade --install argocd argo/argo-cd \
  --namespace argocd \
  --create-namespace \
  --version "${ARGOCD_VERSION}" \
  --values infra/argocd/argocd-values.yml \
  --wait

# Apply xyn-pos ArgoCD Application CRDs
kubectl apply -f infra/argocd/apps/

ok "ArgoCD installed"
echo ""
echo "  ArgoCD UI:  https://argocd.xyn-local.dev  (or port-forward: kubectl port-forward svc/argocd-server -n argocd 8443:443)"
echo "  Admin pass: kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d"

echo ""
echo "==> Post-install complete! xyn-pos infrastructure is ready."
echo "    Next step: push your service manifests and ArgoCD will deploy them automatically."
