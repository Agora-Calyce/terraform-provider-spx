# Terraform SPX Provider

This Terraform provider allows you to interact with the SPX API to manage infrastructure resources such as VPCs, subnets, Elastic IPs (EIPs), and disks.

It talks to the [SPX TF Proxy API](https://github.com/Agora-Calyce/spx-tf-api), which must be reachable at the configured `endpoint`.

> **Transparency Note**: This project was “vibecoded.” It certainly has room for improvement, and all contributions are welcome!

## Installation

To install the provider locally, use the `make build` command. This command compiles the binary and places it in the Terraform plugins directory (`~/.terraform.d/plugins/`).

```bash
make build
```

This will also run a `terraform init` command to load the newly installed plugin.

## Provider Configuration

```hcl
terraform {
  required_providers {
    spx = {
      source  = "agora-calyce/spx"
      version = "1.0.0"
    }
  }
}

provider "spx" {
  endpoint   = "http://localhost:8080"
  api_token  = "your-api-token"
  az         = "az-name"
  orga_id    = "your-orga-id"
  project_id = "your-project-id"
}
```

### Environment Variables

You can also configure the provider using environment variables:

| Variable | Description |
| --- | --- |
| `SPX_API_ENDPOINT` | API endpoint (default: `http://localhost:8080`) |
| `SPX_API_TOKEN` | API token for authentication |
| `SPX_AZ` | Availability Zone |
| `SPX_ORGA_ID` | Organization ID |
| `SPX_PROJECT_ID` | Project ID |

## Examples

The `examples` folder has multiple terraform infrastructures.

## Releasing

Pushing a `vX.Y.Z` tag triggers the `Release` workflow: GoReleaser builds the binaries for every platform, signs the checksums with GPG and publishes a GitHub release that the Terraform Registry picks up.

It needs two repository secrets: `GPG_PRIVATE_KEY` (ASCII-armored private key, whose public key is registered in the Terraform Registry) and `PASSPHRASE`.
