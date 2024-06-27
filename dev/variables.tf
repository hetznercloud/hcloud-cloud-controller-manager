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
