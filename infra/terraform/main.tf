# xyn-pos-v1 — Hetzner Cloud Infrastructure
# Terraform 1.15.5
#
# Topology:
#   - 1x VPS dev+staging  (cx32:  8 vCPU, 16 GB RAM)   → xyn-dev + xyn-staging namespaces
#   - 1x VPS production   (ccx23: 4 vCPU dedicated, 16 GB RAM) → xyn-prod namespace
#   - Shared SSH key, firewall rules, private network
#   - Cloudflare DNS for all subdomains

# ─────────────────────────────────────────────────────────
# SSH Key
# ─────────────────────────────────────────────────────────
resource "hcloud_ssh_key" "xyn_key" {
  name       = "xyn-pos-key"
  public_key = file(var.ssh_public_key_path)
}

# ─────────────────────────────────────────────────────────
# Private Network (for internal communication)
# ─────────────────────────────────────────────────────────
resource "hcloud_network" "xyn_net" {
  name     = "xyn-pos-network"
  ip_range = "10.0.0.0/16"
}

resource "hcloud_network_subnet" "xyn_subnet" {
  network_id   = hcloud_network.xyn_net.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = "10.0.1.0/24"
}

# ─────────────────────────────────────────────────────────
# Firewall
# ─────────────────────────────────────────────────────────
resource "hcloud_firewall" "xyn_firewall" {
  name = "xyn-pos-firewall"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["0.0.0.0/0", "::/0"]
    description = "SSH"
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
    description = "HTTP (redirect to HTTPS)"
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
    description = "HTTPS"
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "6443"
    source_ips = ["0.0.0.0/0", "::/0"]
    description = "K3s API server"
  }

  # Block all other inbound traffic
  rule {
    direction  = "in"
    protocol   = "icmp"
    source_ips = ["0.0.0.0/0", "::/0"]
    description = "ICMP ping"
  }
}

# ─────────────────────────────────────────────────────────
# cloud-init script — installed on both servers
# ─────────────────────────────────────────────────────────
locals {
  cloud_init = <<-EOT
    #cloud-config
    package_update: true
    package_upgrade: true
    packages:
      - curl
      - wget
      - git
      - htop
      - ufw
      - fail2ban
      - unattended-upgrades

    runcmd:
      # UFW firewall
      - ufw default deny incoming
      - ufw default allow outgoing
      - ufw allow 22/tcp
      - ufw allow 80/tcp
      - ufw allow 443/tcp
      - ufw allow 6443/tcp
      - ufw --force enable

      # fail2ban for SSH brute force protection
      - systemctl enable fail2ban
      - systemctl start fail2ban

      # Swap (improves K3s stability on 16GB servers)
      - fallocate -l 4G /swapfile
      - chmod 600 /swapfile
      - mkswap /swapfile
      - swapon /swapfile
      - echo '/swapfile none swap sw 0 0' >> /etc/fstab

      # K3s kernel parameters
      - modprobe br_netfilter
      - sysctl -w net.bridge.bridge-nf-call-iptables=1
      - sysctl -w net.bridge.bridge-nf-call-ip6tables=1
      - sysctl -w net.ipv4.ip_forward=1
      - echo "net.bridge.bridge-nf-call-iptables=1" >> /etc/sysctl.d/k3s.conf
      - echo "net.ipv4.ip_forward=1" >> /etc/sysctl.d/k3s.conf
  EOT
}

# ─────────────────────────────────────────────────────────
# Dev + Staging Server (cx32)
# ─────────────────────────────────────────────────────────
resource "hcloud_server" "dev_staging" {
  name        = "xyn-pos-dev-staging"
  image       = var.image
  server_type = var.server_type_dev_staging
  location    = var.location
  ssh_keys    = [hcloud_ssh_key.xyn_key.id]
  firewall_ids = [hcloud_firewall.xyn_firewall.id]
  user_data   = local.cloud_init

  labels = {
    environment = "dev-staging"
    project     = "xyn-pos"
    managed-by  = "terraform"
  }

  network {
    network_id = hcloud_network.xyn_net.id
    ip         = "10.0.1.10"
  }

  lifecycle {
    prevent_destroy = false   # Set to true in production
  }
}

# ─────────────────────────────────────────────────────────
# Production Server (ccx23 — dedicated vCPU)
# ─────────────────────────────────────────────────────────
resource "hcloud_server" "prod" {
  name        = "xyn-pos-prod"
  image       = var.image
  server_type = var.server_type_prod
  location    = var.location
  ssh_keys    = [hcloud_ssh_key.xyn_key.id]
  firewall_ids = [hcloud_firewall.xyn_firewall.id]
  user_data   = local.cloud_init

  labels = {
    environment = "production"
    project     = "xyn-pos"
    managed-by  = "terraform"
  }

  network {
    network_id = hcloud_network.xyn_net.id
    ip         = "10.0.1.20"
  }

  lifecycle {
    prevent_destroy = true   # Prevent accidental destroy in prod
  }
}

# ─────────────────────────────────────────────────────────
# Backups for production server
# ─────────────────────────────────────────────────────────
resource "hcloud_server_backup" "prod_backup" {
  server_id = hcloud_server.prod.id
}

# ─────────────────────────────────────────────────────────
# Volumes for production (additional persistent storage)
# ─────────────────────────────────────────────────────────
resource "hcloud_volume" "prod_data" {
  name      = "xyn-pos-prod-data"
  size      = 200   # 200 GB
  server_id = hcloud_server.prod.id
  automount = true
  format    = "ext4"

  labels = {
    environment = "production"
    project     = "xyn-pos"
  }
}

# ─────────────────────────────────────────────────────────
# Cloudflare DNS Records
# ─────────────────────────────────────────────────────────

# Dev + Staging
resource "cloudflare_record" "dev_wildcard" {
  zone_id = var.cloudflare_zone_id
  name    = "*.dev.${var.domain}"
  content   = hcloud_server.dev_staging.ipv4_address
  type    = "A"
  ttl     = 300
  proxied = false   # Direct DNS for K3s TLS termination
}

resource "cloudflare_record" "staging_wildcard" {
  zone_id = var.cloudflare_zone_id
  name    = "*.staging.${var.domain}"
  content   = hcloud_server.dev_staging.ipv4_address
  type    = "A"
  ttl     = 300
  proxied = false
}

# Production
resource "cloudflare_record" "prod_root" {
  zone_id = var.cloudflare_zone_id
  name    = var.domain
  content   = hcloud_server.prod.ipv4_address
  type    = "A"
  ttl     = 60
  proxied = true   # Cloudflare proxy (DDoS protection) for prod
}

resource "cloudflare_record" "prod_wildcard" {
  zone_id = var.cloudflare_zone_id
  name    = "*.${var.domain}"
  content   = hcloud_server.prod.ipv4_address
  type    = "A"
  ttl     = 60
  proxied = true
}

# ArgoCD subdomains
resource "cloudflare_record" "argocd_dev" {
  zone_id = var.cloudflare_zone_id
  name    = "argocd.dev.${var.domain}"
  content   = hcloud_server.dev_staging.ipv4_address
  type    = "A"
  ttl     = 300
  proxied = false
}

resource "cloudflare_record" "argocd_prod" {
  zone_id = var.cloudflare_zone_id
  name    = "argocd.${var.domain}"
  content   = hcloud_server.prod.ipv4_address
  type    = "A"
  ttl     = 300
  proxied = false   # ArgoCD UI: bypass Cloudflare proxy
}
