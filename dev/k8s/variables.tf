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
variable "robot_user" {
  type      = string
  sensitive = true
}
variable "robot_password" {
  type      = string
  sensitive = true
}
