# System Testing

Some scripts to test the OCI Flexvolume driver using a real Kubernetes cluster.

## Usage

We first need to setup the environment. The following must be defined:

* $KUBECONFIG or $KUBECONFIG\_VAR

If --create-using-oci flag is set, then the following will
need to be defined:

* $OCI\_API\_KEY or $OCI\_API\_KEY\_VAR

Note: If set, OCI\_API\_KEY/KUBECONFIG must contain the path to the required
files. Alternatively, OCI\_API\_KEY\_VAR/KUBECONFIG\_VAR must contain the content
of the required files (base64 encoded). If both are set, the former will
take precedence.

If the --enforce-cluster-locking flag is set, then the following will 
need to be defined:

* $INSTANCE\_KEY or $INSTANCE\_KEY\_VAR
* $MASTER\_IP
* $SLAVE0\_IP
* $SLAVE1\_IP
* $WERCKER\_API\_TOKEN

If the --install flag is set, then the following will 
need to be defined:

* $MASTER\_IP
* $SLAVE0\_IP
* $SLAVE1\_IP
* $VCN

We can then run the system test as follows:

```
cd test/system
./runnner.py
```

Note: This will provision a new test volume. If we want to run the tests and keep this test volume 
around for future runs, then execute the following:

```
cd test/integration
./runnner.py --no-destroy
```

Then if we want to run the tests on a volume that already exists (i.e. skip the provisioning step),
execute the following:

```
cd test/integration
./runnner.py --no-create
```

