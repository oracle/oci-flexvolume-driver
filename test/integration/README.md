# OCI Flexvolume driver integration tests

This suite of tests is compiled by `make build-integration-tests` into a binary
that is then uploaded to an OCI compute instance for execution.

The tests perform real attach/detach and mount/unmount operations in the cloud,
however, do not call the driver binary directly.

## Configuration

The tests are configured to provision an instance and volume in the bristoldev
tenancy, however, keys for access to this environment must be provided via
environment variables.

 - `$OCI_API_KEY`: base64 encoded OCI API signing key.
 - `$INSTANCE_KEY`: base64 encoded SSH private key corresponding to
   `$INSTANCE_KEY_PUB`.
 - `$INSTANCE_KEY_PUB`: base64 encoded SSH public key (added to instance
   authorized keys).

## Usage

To provision the required resources and run the full suite of tests (presuming
the aforementioned environment variables are set) do the following:

```bash
$ cd test/integration
$ ./run.sh
```

When running locally it is often useful to leave the cloud infrastructure in
place for detailed inspection.

```bash
$ cd test/integration
$ ./run.sh --no-destroy
$ cd terraform/
$ ssh -i _tmp/instance_key opc@$(terraform output instance_public_ip)
$ # Debug some stuff...
$ export TF_VAR_test_id=$(terraform output test_id)
$ terraform destroy .
```

NOTE: `terraform destroy` only works when executed from
`test/integration/terraform` due to relative file path issues.
