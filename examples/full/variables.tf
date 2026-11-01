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

variable "ssh_public_key_path" {
  description = "Chemin de la clé publique SSH autorisée à se connecter à l'instance"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "admin_cidr" {
  description = "Plage d'IP autorisée en SSH sur l'instance (à restreindre à votre IP publique)"
  type        = string
  default     = "0.0.0.0/0"
}
