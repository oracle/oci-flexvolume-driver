# OCI Flexvolume Driver

[![wercker status](https://app.wercker.com/status/9fdba1db3d8e347756aa19819dddc4d5/s/master "wercker status")](https://app.wercker.com/project/byKey/9fdba1db3d8e347756aa19819dddc4d5)

This project implements a flexvolume driver for Kubernetes clusters
running on Oracle Cloud Infrastructure (OCI). It enables mounting of [OCI block
storage volumes][1] to Kubernetes pods via the [Flexvolume][2] plugin interface.

We recommend you use this driver in conjunction with the OCI Volume Provisioner.
See the [oci-volume-provisioner][3] for more information.

## Install / Setup

See [here](docs/install.md) for information about installing and configuring the volume driver.

## Getting started

See [here](docs/tutorial.md) for a tutorial and guide to using the flexvolume volume driver.

## Support

Please checkout our documentation. If you find a bug, please raise an
[issue](https://github.com/oracle/oci-flexvolume-driver/issues/new)

[1]: https://docs.us-phoenix-1.oraclecloud.com/Content/Block/Concepts/overview.htm
[2]: https://github.com/kubernetes/community/blob/master/contributors/devel/flexvolume.md
[3]: https://github.com/oracle/oci-volume-provisoner
