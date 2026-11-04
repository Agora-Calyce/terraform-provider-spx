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
