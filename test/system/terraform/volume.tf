variable "tenancy_ocid" {
  default = "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq"
}

variable "user_ocid" {
  default = "ocid1.user.oc1..aaaaaaaao235lbcxvdrrqlrpwv4qvil2xzs4544h3lof4go3wz2ett6arpeq"
}

variable "fingerprint" {
  default = "f1:d8:e7:75:8d:3a:81:a0:18:2f:fa:8a:8f:64:44:66"
}

variable "private_key_path" {
  default = "/tmp/oci_api_key.pem"
}

variable "compartment_ocid" {
  default = "ocid1.compartment.oc1..aaaaaaaa6yrzvtwcumheirxtmbrbrya5lqkr7k7lxi34q3egeseqwlq2l5aq"
}

variable "availability_domain" {
  default = "NWuj:PHX-AD-2"
}

variable "region" {
  default = "us-phoenix-1"
}

variable "test_id" {}

provider "oci" {
  tenancy_ocid     = "${var.tenancy_ocid}"
  user_ocid        = "${var.user_ocid}"
  fingerprint      = "${var.fingerprint}"
  private_key_path = "${var.private_key_path}"
  region           = "${var.region}"
}

resource "oci_core_volume" "test_volume" {
  availability_domain = "${var.availability_domain}"
  compartment_id      = "${var.compartment_ocid}"
  display_name        = "flexvolumesystemtest${var.test_id}"
  size_in_gbs         = "50"
}

output "volume_ocid" {
  value = "${oci_core_volume.test_volume.id}"
}
