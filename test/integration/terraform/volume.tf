resource "oci_core_volume" "test_volume" {
  availability_domain = "${var.availability_domain}"
  compartment_id = "${var.compartment_ocid}"
  display_name = "${var.flexvolume_test_id}"
  size_in_mbs = "51200"  # 50GB
}

output "volume_ocid" {
  value = "${oci_core_volume.test_volume.id}"
}
