output "dev_staging_ip" {
  description = "Public IP of the dev+staging server"
  value       = hcloud_server.dev_staging.ipv4_address
}

output "prod_ip" {
  description = "Public IP of the production server"
  value       = hcloud_server.prod.ipv4_address
  sensitive   = false
}

output "dev_staging_ssh" {
  description = "SSH command for dev+staging server"
  value       = "ssh root@${hcloud_server.dev_staging.ipv4_address}"
}

output "prod_ssh" {
  description = "SSH command for production server"
  value       = "ssh root@${hcloud_server.prod.ipv4_address}"
}

output "prod_volume_id" {
  description = "Hetzner volume ID for production data disk"
  value       = hcloud_volume.prod_data.id
}

output "next_steps" {
  description = "Post-provisioning steps"
  value       = <<-EOT
    ✅ Infrastructure provisioned. Next steps:

    1. SSH to dev+staging: ${hcloud_server.dev_staging.ipv4_address}
       ssh root@${hcloud_server.dev_staging.ipv4_address}
       ./scripts/k3s/install.sh
       ./scripts/k3s/post-install.sh

    2. SSH to production:
       ssh root@${hcloud_server.prod.ipv4_address}
       ./scripts/k3s/install.sh
       ./scripts/k3s/post-install.sh

    3. Apply ArgoCD App-of-Apps to both clusters:
       kubectl apply -f infra/argocd/apps/app-of-apps.yml -n argocd

    DNS records are live — cert-manager will issue Let's Encrypt certs automatically.
  EOT
}
