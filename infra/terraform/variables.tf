variable "hcloud_token" {
  description = "Hetzner Cloud API token (generate at console.hetzner.cloud)"
  type        = string
  sensitive   = true
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token for DNS management"
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare Zone ID for your domain"
  type        = string
}

variable "domain" {
  description = "Root domain for xyn-pos (e.g. xyn-pos.dev)"
  type        = string
  default     = "xyn-pos.dev"
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key for server access"
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Environment must be 'staging' or 'production'."
  }
}

variable "server_type_dev_staging" {
  description = "Hetzner server type for dev+staging VPS"
  type        = string
  default     = "cx32"   # 8 vCPU, 16 GB RAM, 160 GB SSD — ~€18.19/month
}

variable "server_type_prod" {
  description = "Hetzner server type for production VPS"
  type        = string
  default     = "ccx23"  # 4 vCPU (dedicated), 16 GB RAM, 160 GB SSD — ~€58.76/month
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "hel1"   # Helsinki — best latency for SEA
  # Options: nbg1 (Nuremberg), fsn1 (Falkenstein), hel1 (Helsinki), ash (Ashburn US), hil (Hillsboro US), sin (Singapore)
}

variable "image" {
  description = "Server OS image"
  type        = string
  default     = "ubuntu-24.04"
}
