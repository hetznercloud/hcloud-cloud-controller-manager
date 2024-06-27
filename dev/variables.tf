variable "name" {
  type = string
}
variable "hcloud_token" {
  type      = string
  sensitive = true
}
variable "k3s_channel" {
  type = string
}
variable "setup_robot" {
  type    = bool
  default = false
}
# Hetzner Robot
variable "robot_user" {
  type      = string
  sensitive = true
}
variable "robot_password" {
  type      = string
  sensitive = true
}