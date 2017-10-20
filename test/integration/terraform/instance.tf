variable "ssh_public_key" {
  default = ""
}

variable "ssh_private_key" {
  default = ""
}

variable subnet_ocid {
  default = "ocid1.subnet.oc1.phx.aaaaaaaahuxrgvs65iwdz7ekwgg3l5gyah7ww5klkwjcso74u3e4i64hvtvq"
}

# Gets the OCID of the OS image to use
data "oci_core_images" "os_image_ocid" {
    compartment_id = "${var.compartment_ocid}"
    operating_system = "Oracle Linux"
    operating_system_version = "7.3"
}

resource "oci_core_instance" "instance" {
  availability_domain = "${var.availability_domain}"
  compartment_id = "${var.compartment_ocid}"
  display_name = "${var.flexvolume_test_id}"
  image = "${lookup(data.oci_core_images.os_image_ocid.images[0], "id")}"
  shape = "VM.Standard1.1"
  subnet_id =  "${var.subnet_ocid}"
  metadata {
    ssh_authorized_keys = "${var.ssh_public_key}"
  }
  timeouts {
    create = "60m"
  }
}

data "oci_core_vnic_attachments" "instance_vnics" {
  compartment_id = "${var.compartment_ocid}"
  availability_domain = "${var.availability_domain}"
  instance_id = "${oci_core_instance.instance.id}"
}

# Gets the OCID of the first (default) vNIC
data "oci_core_vnic" "instance_vnic" {
  vnic_id = "${lookup(data.oci_core_vnic_attachments.instance_vnics.vnic_attachments[0], "vnic_id")}"
}

resource null_resource "instance" {
  depends_on = [
    "data.oci_core_vnic.instance_vnic",
  ]

  triggers {
     instance_id = "${oci_core_instance.instance.id}"
  }

  connection {
    type = "ssh"
    host = "${data.oci_core_vnic.instance_vnic.public_ip_address}"
    user = "opc"
    private_key = "${var.ssh_private_key}"
  }

  provisioner "file" {
    source      = "flexvolume_driver.json",
    destination = "/tmp/flexvolume_driver.json",
  }

  provisioner "file" {
    source      = "_tmp/oci_api_key.pem",
    destination = "/tmp/oci_api_key.pem",
  }

  provisioner "file" {
    source      = "../../../dist/bin/integration-tests",
    destination = "/tmp/integration-tests",
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/integration-tests",
    ]
  }
}

output "instance_public_ip" {
  value = "${data.oci_core_vnic.instance_vnic.public_ip_address}"
}
