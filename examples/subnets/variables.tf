variable "endpoint" {
  description = "L'URL de l'API"
  type        = string
  default     = "http://localhost:8080"
}

variable "api_token" {
  description = "Le token d'API pour l'authentification"
  type        = string
  sensitive   = true
}

variable "orga_id" {
  description = "L'ID de l'organisation"
  type        = string
}

variable "az" {
  description = "La zone de disponibilité (AZ)"
  type        = string
}

variable "project_id" {
  description = "L'ID du projet"
  type        = string
}
