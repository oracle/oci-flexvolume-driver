# System Testing

Some scripts to test the OCI Flexvolume driver using a real Kubernetes cluster.

## Usage

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

