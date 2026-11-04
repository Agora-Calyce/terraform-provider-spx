variable "endpoint" {
  description = "The API URL"
  type        = string
  default     = "http://localhost:8080"
}

variable "api_token" {
  description = "The API token used for authentication"
  type        = string
  sensitive   = true
}

variable "orga_id" {
  description = "The organization ID"
  type        = string
}

variable "az" {
  description = "The availability zone (AZ)"
  type        = string
}

variable "project_id" {
  description = "The project ID"
  type        = string
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key allowed to connect to the instance"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "admin_cidr" {
  description = "IP range allowed to SSH into the instance (restrict it to your public IP)"
  type        = string
  default     = "0.0.0.0/0"
}
