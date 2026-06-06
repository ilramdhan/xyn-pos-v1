terraform {
  required_version = ">= 1.15.5"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.50"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
  }

  # Remote state — use Terraform Cloud or S3-compatible (MinIO)
  # Uncomment and configure for team use:
  # backend "s3" {
  #   bucket                      = "xyn-terraform-state"
  #   key                         = "xyn-pos-v1/terraform.tfstate"
  #   region                      = "main"
  #   endpoint                    = "https://your-minio-endpoint:9000"
  #   access_key                  = var.minio_access_key
  #   secret_key                  = var.minio_secret_key
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   force_path_style            = true
  # }
}

provider "hcloud" {
  token = var.hcloud_token
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
