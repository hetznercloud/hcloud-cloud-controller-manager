# General
variable "name" {
  type = string
}
variable "k3s_channel" {
  type = string
}

# Hetzner Cloud
variable "hcloud_token" {
  type      = string
  sensitive = true
}

# Hetzner Robot
variable "robot_enabled" {
  type    = bool
  default = false
}
