#!/usr/bin/env python

# Copyright 2017 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import atexit
import datetime
import json
import os
import re
import select
import shutil
import subprocess
import sys
import time
import urllib2
import uuid


TMP_KUBECONFIG = "/tmp/kubeconfig"
TMP_OCI_API_KEY = "/tmp/oci_api_key.pem"
TMP_INSTANCE_KEY = "/tmp/instance_key"
DEBUG_FILE = "runner.log"
DRIVER_DIR = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/oracle~oci"
TERRAFORM_DIR = "terraform"
TIMEOUT = 180
LOCKFILE = "/tmp/system-test-lock-file"
MAX_NUM_LOCKFILE_RETRIES = 100
CI_LOCKFILE_PREFIX = "CI"
LOCAL_LOCKFILE_PREFIX = "LOCAL"
DAEMONSET_NAME = "oci-flexvolume-driver"
CI_APPLICATION_NAME = "oci-flexvolume-driver"
CI_BASE_URL = "https://app.wercker.com/api/v3"
CI_PIPELINE_NAME = "system-test"
WRITE_REPORT=True
REPORT_DIR_PATH="/tmp/results"
REPORT_FILE="done"

# On exit return 0 for success or any other integer for a failure.
# If write_report is true then write a completion file to the Sonabuoy plugin result file.
# The default location is: /tmp/results/done
def _finish_with_exit_code(exit_code, write_report=True, report_dir_path=REPORT_DIR_PATH, report_file=REPORT_FILE):
    if write_report:
        if os.path.exists(report_dir_path):
            print "deleting report_dir_path: " + report_dir_path
            shutil.rmtree(report_dir_path) 
        os.makedirs(report_dir_path)
        print "created file report_dir_path: " + report_dir_path
        with open(report_dir_path + "/" + report_file, "w+") as file: 
            file.write(str(exit_code))
    sys.exit(exit_code)        

def _check_env(args):
    should_exit = False
    if args['create_using_oci']:
        if "OCI_API_KEY" not in os.environ and "OCI_API_KEY_VAR" not in os.environ:
            _log("Error. Can't find either OCI_API_KEY or OCI_API_KEY_VAR in the environment.")
            should_exit = True
    if args['enforce_cluster_locking']:
        if "INSTANCE_KEY" not in os.environ and "INSTANCE_KEY_VAR" not in os.environ:
            _log("Error. Can't find either INSTANCE_KEY or INSTANCE_KEY_VAR in the environment.")
            should_exit = True
    if args['enforce_cluster_locking'] or args['install']:
        if "MASTER_IP" not in os.environ:
            _log("Error. Can't find MASTER_IP in the environment.")
            should_exit = True
        if "SLAVE0_IP" not in os.environ:
            _log("Error. Can't find SLAVE0_IP in the environment.")
            should_exit = True
        if "SLAVE1_IP" not in os.environ:
            _log("Error. Can't find SLAVE1_IP in the environment.")
            should_exit = True
        if "VCN" not in os.environ:
            _log("Error. Can't find VCN in the environment.")
            should_exit = True
    if args['enforce_cluster_locking']:
        if "WERCKER_API_TOKEN" not in os.environ:
            _log("Error. Can't find WERCKER_API_TOKEN in the environment.")
            should_exit = True

    if should_exit:
        _finish_with_exit_code(1)


def _create_key_files():
    if "KUBECONFIG_VAR" in os.environ:
        _run_command("echo \"$KUBECONFIG_VAR\" | openssl enc -base64 -d -A > " + TMP_KUBECONFIG, ".")
    if "OCI_API_KEY_VAR" in os.environ:
        _run_command("echo \"$OCI_API_KEY_VAR\" | openssl enc -base64 -d -A > " + TMP_OCI_API_KEY, ".")
        _run_command("chmod 600 " + TMP_OCI_API_KEY, ".")
    if "INSTANCE_KEY_VAR" in os.environ:
        _run_command("echo \"$INSTANCE_KEY_VAR\" | openssl enc -base64 -d -A > " + TMP_INSTANCE_KEY, ".")
        _run_command("chmod 600 " + TMP_INSTANCE_KEY, ".")


def _read_lock_file(instance_ip):
    stdout, _, returncode = _run_command(_ssh(instance_ip, "cat " + LOCKFILE), ".", display_errors=False)
    if returncode == 0:
        return stdout.strip()
    return None


def _get_lockfile_content(test_uuid):
    if 'WERCKER' in os.environ:
        return CI_LOCKFILE_PREFIX + "-" + test_uuid
    else:
        return LOCAL_LOCKFILE_PREFIX + "-" + test_uuid


def _started_from_ci(content):
    return re.match(CI_LOCKFILE_PREFIX + "-.*", content)


def _write_lock_file(instance_ip, content):
    _log("Creating lockfile: " + LOCKFILE)
    _, _, returncode = _run_command(_ssh(instance_ip, "echo " + content + " > " + LOCKFILE), ".")
    return returncode == 0


def _delete_lock_file(instance_ip):
    _log("Deleting lockfile: " + LOCKFILE)
    _, _, returncode = _run_command(_ssh(instance_ip, "rm -rf " + LOCKFILE), ".")
    return returncode == 0


def _get_url(url):
    _log("Querying URL: " + url)
    headers = {'Authorization': 'Bearer ' + os.environ['WERCKER_API_TOKEN'],
               'Content-Type': 'application/json'}
    request = urllib2.Request(url, None, headers)
    response = urllib2.urlopen(request)
    return response.read()


def _get_application_id(applications_json):
    for application in json.loads(applications_json):
        if application['name'] == CI_APPLICATION_NAME:
            return application['id']
    _log("Error. Failed to find the CI application id for: " + CI_APPLICATION_NAME)
    _finish_with_exit_code(1)


def _pipeline_exists(runs_json):
    for run in json.loads(runs_json):
        if run['pipeline']['name'] == CI_PIPELINE_NAME:
            if 'WERCKER_RUN_ID' in os.environ:
                # We are running from within CI so we better check that the run found is not us!
                if os.environ['WERCKER_RUN_ID'] != run['id']:
                    return True
            else:
                # Not running from CI, so any running pipeline must be left alone to complete.
                return True
    return False


def _instance_running_in_ci():
    applications_json = _get_url(CI_BASE_URL + "/applications/oracle")
    application_id = _get_application_id(applications_json)
    runs_json = _get_url(CI_BASE_URL + "/runs?applicationId=" + application_id + "&status=running")
    return _pipeline_exists(runs_json)


def _wait_for_cluster(instance_ip):
    test_uuid = str(uuid.uuid4())
    lockfile_content = _get_lockfile_content(test_uuid)
    i = 0
    while i < MAX_NUM_LOCKFILE_RETRIES:
        content = _read_lock_file(instance_ip)
        if content is None:
            # No lockfile found, so create one.
            if not _write_lock_file(instance_ip, lockfile_content):
                _log("Error. Failed to write lockfile: " + LOCKFILE)
                _finish_with_exit_code(1)
            content = _read_lock_file(instance_ip)
            if content == lockfile_content:
                return
        else:
            # Lockfile found, so check there really is an instance of this pipeline running,
            # and if not delete it. Note: This is to work around the issue of a previous
            # cancelled CI run leaving a dangling lockfile.
            if _started_from_ci(content) and not _instance_running_in_ci():
                _log("Dangling lockfile detected, deleting it...")
                _delete_lock_file(instance_ip)
                continue
        time.sleep(30)
    _log("Error. Timedout waiting for the cluster to become available")
    _finish_with_exit_code(1)


def _destroy_key_files():
    if "KUBECONFIG_VAR" in os.environ:
        os.remove(TMP_KUBECONFIG)
    if "OCI_API_KEY_VAR" in os.environ:
        os.remove(TMP_OCI_API_KEY)
    if "INSTANCE_KEY_VAR" in os.environ:
        os.remove(TMP_INSTANCE_KEY)


def _get_kubeconfig_file():
    return os.environ['KUBECONFIG'] if "KUBECONFIG" in os.environ else TMP_KUBECONFIG


def _get_instance_key_file():
    return os.environ['INSTANCE_KEY'] if "INSTANCE_KEY" in os.environ else TMP_INSTANCE_KEY


def _banner(as_banner, bold):
    if as_banner:
        if bold:
            print "********************************************************"
        else:
            print "--------------------------------------------------------"


def _reset_debug_file():
    if os.path.exists(DEBUG_FILE):
        os.remove(DEBUG_FILE)


def _debug_file(string):
    with open(DEBUG_FILE, "a") as debug_file:
        debug_file.write(string)


def _log(string, as_banner=False, bold=False):
    _banner(as_banner, bold)
    print string
    _banner(as_banner, bold)
    sys.stdout.flush()


def _process_stream(stream, read_fds, global_buf, line_buf):
    char = stream.read(1)
    if char == '':
        read_fds.remove(stream)
    global_buf.append(char)
    line_buf.append(char)
    if char == '\n':
        _debug_file(''.join(line_buf))
        line_buf = []
    return line_buf


def _poll(stdout, stderr):
    stdoutbuf = []
    stdoutbuf_line = []
    stderrbuf = []
    stderrbuf_line = []
    read_fds = [stdout, stderr]
    x_fds = [stdout, stderr]
    while read_fds:
        rlist, _, _ = select.select(read_fds, [], x_fds)
        if rlist:
            for stream in rlist:
                if stream == stdout:
                    stdoutbuf_line = _process_stream(stream, read_fds, stdoutbuf, stdoutbuf_line)
                if stream == stderr:
                    stderrbuf_line = _process_stream(stream, read_fds, stderrbuf, stderrbuf_line)
    return (''.join(stdoutbuf), ''.join(stderrbuf))


def _run_command(cmd, cwd, display_errors=True):
    _log(cwd + ": " + cmd)
    process = subprocess.Popen(cmd,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               shell=True, cwd=cwd)
    (stdout, stderr) = _poll(process.stdout, process.stderr)
    returncode = process.wait()
    if returncode != 0 and display_errors:
        _log("    stdout: " + stdout)
        _log("    stderr: " + stderr)
        _log("    result: " + str(returncode))
    return (stdout, stderr, returncode)


def _get_terraform_env():
    timestamp = datetime.datetime.now().strftime('%Y%m%d%H%M%S%f')
    return "TF_VAR_test_id=" + timestamp


def _terraform(action, cwd, terraform_env):
    (stdout, _, returncode) = _run_command(terraform_env + " terraform " + action, cwd)
    if returncode != 0:
        _log("Error running terraform")
        _finish_with_exit_code(1)
    return stdout


def _get_cluster_ips():
    return os.environ['MASTER_IP'], [os.environ['SLAVE0_IP'], os.environ['SLAVE1_IP']]


def _kubectl(action, exit_on_error=True, display_errors=True, log_stdout=True):
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        (stdout, _, returncode) = _run_command("kubectl " + action, ".", display_errors)
    else:
        (stdout, _, returncode) = _run_command("KUBECONFIG=" + _get_kubeconfig_file() + \
                " kubectl " + action, ".", display_errors)
    if exit_on_error and returncode != 0:
        _log("Error running kubectl")
        _finish_with_exit_code(1)
    if log_stdout:
        _log(stdout)
    return stdout


def _ssh(instance_ip, cmd):
    return "ssh -o UserKnownHostsFile=/dev/null " + \
           "-o LogLevel=quiet " + \
           "-o StrictHostKeyChecking=no " + \
           "-i " + _get_instance_key_file() + " opc@" + instance_ip + " " + \
           "\"bash --login -c \'" + cmd + "\'\""


def _patch_template_file(infile, outfile, volume_name, test_id):
    with open(infile, "r") as sources:
        lines = sources.readlines()
    with open(outfile + "." + test_id, "w") as sources:
        for line in lines:
            patched_line = line
            if volume_name is not None:
                patched_line = re.sub('{{VOLUME_NAME}}', volume_name, patched_line)
            patched_line = re.sub('{{TEST_ID}}', test_id, patched_line)
            sources.write(patched_line)
    return outfile + "." + test_id


def _create_replication_controller_yaml(using_oci, volume_name, test_id):
    if using_oci:
        return _patch_template_file(
            "replication-controller.yaml.template",
            "replication-controller.yaml",
            volume_name, test_id)
    else:
        return _patch_template_file(
            "replication-controller-with-volume-claim.yaml.template",
            "replication-controller-with-volume-claim.yaml",
            volume_name, test_id)


def _is_driver_running():
    stdout = _kubectl("-n kube-system get daemonset " + DAEMONSET_NAME + " -o json", log_stdout=False)
    jsn = json.loads(stdout)
    desired = int(jsn["status"]["desiredNumberScheduled"])
    ready = int(jsn["status"]["numberReady"])
    _log("    - daemonset " + DAEMONSET_NAME + ": desired: " + str(desired) + ", ready: " + str(ready))
    return desired == ready


def _wait_for_driver():
    num_polls = 0
    while not _is_driver_running():
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            _log("Error: Daemonset: " + DAEMONSET_NAME + " " + "failed to achieve running status: ")
            _finish_with_exit_code(1)


def _install_driver():
    _kubectl("delete -f ../../dist/oci-flexvolume-driver.yaml", exit_on_error=False, display_errors=False)
    _kubectl("apply -f ../../dist/oci-flexvolume-driver.yaml")
    _wait_for_driver()


def _get_pod_infos(test_id):
    stdout = _kubectl("get pods -o wide", log_stdout=False)
    infos = []
    for line in stdout.split("\n"):
        line_array = line.split()
        if re.match(r"nginx-controller-" + test_id + ".*", line):
            name = line_array[0]
            status = line_array[2]
            node = line_array[6]
            infos.append((name, status, node))
    return infos


def _wait_for_pod_status(desired_status, test_id):
    infos = _get_pod_infos(test_id)
    num_polls = 0
    while not any(i[1] == desired_status for i in infos):
        for i in infos:
            _log("    - pod: " + i[0] + ", status: " + i[1] + ", node: " + i[2])
        time.sleep(1)
        num_polls += 1
        if num_polls == TIMEOUT:
            for i in infos:
                _log("Error: Pod: " + i[0] + " " +
                     "failed to achieve status: " + desired_status + "." +
                     "Final status was: " + i[1])
            _finish_with_exit_code(1)
        infos = _get_pod_infos(test_id)
    for i in infos:
        if i[1] == desired_status:
            return (i[0], i[1], i[2])
    # Should never get here.
    return (None, None, None)


def _get_volume_name(terraform_env):
    output = _terraform("output -json", TERRAFORM_DIR, terraform_env)
    jsn = json.loads(output)
    ocid = jsn["volume_ocid"]["value"].split('.')
    return ocid[len(ocid) - 1]


def _cluster_check():
    availabilityDomains = []
    nodes_json = _kubectl("get nodes -o json", log_stdout=False)
    nodes = json.loads(nodes_json)
    for node in nodes['items']:
        availabilityDomains.append(node['metadata']['labels']['failure-domain.beta.kubernetes.io/zone'])
    if len(set(availabilityDomains)) != 1:
        _log("Error: This test requires a cluster with a single region")
        _finish_with_exit_code(1)
    if len(availabilityDomains) < 2:
        _log("Error: This test requires a cluster with at least 2 instances")
        _finish_with_exit_code(1)


def _main():
    _reset_debug_file()
    parser = argparse.ArgumentParser(description='System test runner for the OCI Block Volume flexvolume driver')
    parser.add_argument('--cluster-check',
                        help='Enable the check that tests if the cluster has the correct shape to run this test',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-create',
                        help='Disable the creation of the test volume',
                        action='store_true',
                        default=False)
    parser.add_argument('--create-using-oci',
                        help='If we are creating the test volume, then create it directly via OCI (i.e. dont use the volume provisioner)',
                        action='store_true',
                        default=False)
    parser.add_argument('--enforce-cluster-locking',
                        help='Enforce cluster locking such that only one instance of the test can be run at once',
                        action='store_true',
                        default=False)
    parser.add_argument('--install',
                        help='Install the flexvolume driver in the cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-test',
                        help='Dont run the tests on the test cluster',
                        action='store_true',
                        default=False)
    parser.add_argument('--destructive',
                        help='Run the tests in destructive mode (i.e. nodes are cordoned/uncordoned)',
                        action='store_true',
                        default=False)
    parser.add_argument('--no-destroy',
                        help='If we are creating the test volume, then dont destroy it',
                        action='store_true',
                        default=False)
    args = vars(parser.parse_args())

    _check_env(args)
    _create_key_files()
    atexit.register(_destroy_key_files)

    test_id = str(uuid.uuid4())[:8]

    if args['cluster_check']:
        _cluster_check()

    if args['enforce_cluster_locking']:
        _log("Waiting for the cluster to be available", as_banner=True)
        (master_ip, _) = _get_cluster_ips()
        _wait_for_cluster(master_ip)
        def _delete_lock_file_atexit():
            if not _delete_lock_file(master_ip):
                _log("Error. Failed to delete lockfile: " + LOCKFILE)
                _finish_with_exit_code(1)
        atexit.register(_delete_lock_file_atexit)

    terraform_env = _get_terraform_env()

    if not args['no_create']:
        if args['create_using_oci']:
            _log("Creating test volume (using terraform)", as_banner=True)
            _terraform("init", TERRAFORM_DIR, terraform_env)
            _terraform("apply -auto-approve", TERRAFORM_DIR, terraform_env)
            _log(_terraform("output -json", TERRAFORM_DIR, terraform_env))

    if not args['no_destroy']:
        def _destroy_test_volume_atexit():
            if args['create_using_oci']:
                _log("Destroying test volume (using terraform)", as_banner=True)
                _terraform("destroy -force", TERRAFORM_DIR, terraform_env)
        atexit.register(_destroy_test_volume_atexit)

    if args['create_using_oci']:
        replication_controller = _create_replication_controller_yaml(
            True, _get_volume_name(terraform_env), test_id)
    else:
        replication_controller = _create_replication_controller_yaml(
            False, None, test_id)

    if args['install']:
        _log("Installing flexvolume driver on all of the the nodes", as_banner=True)
        _install_driver()

    if not args['no_test']:
        _log("Running system test: ", as_banner=True)

        _log("Starting the replication controller (creates a single nginx pod).")
        _kubectl("delete -f " + replication_controller, exit_on_error=False, display_errors=False)
        _kubectl("create -f " + replication_controller)

        _log("Waiting for the pod to start.")
        (podname1, _, node1) = _wait_for_pod_status("Running", test_id)

        _log("Writing a file to the flexvolume mounted in the pod.")
        _kubectl("exec " + podname1 + " -- touch /usr/share/nginx/html/hello.txt")

        _log("Does the new file exist?")
        stdout = _kubectl("exec " + podname1 + " -- ls /usr/share/nginx/html")
        if "hello.txt" not in stdout.split("\n"):
            _log("Error: Failed to find file hello.txt in mounted volume")
            _finish_with_exit_code(1)
        _log("Yes it does!")

        if args['destructive']:
            _log("Marking the current node as unschedulable.")
            _kubectl("cordon " + node1)

        _log("Deleting the pod. This should cause it to be restarted (possibly on another node).")
        _kubectl("delete pod " + podname1)

        _log("Waiting for the pod to start (possibly on the other node).")
        (podname2, _, node2) = _wait_for_pod_status("Running", test_id)

        if args['destructive']:
            if node1 == node2:
                _log("Error: Pod failed to appear on the other node after being deleted/restarted.")
                _finish_with_exit_code(1)

        _log("Does the new file still exist?")
        stdout = _kubectl("exec " + podname2 + " -- ls /usr/share/nginx/html")
        if "hello.txt" not in stdout.split("\n"):
            _log("Error: Failed to find file hello.txt in mounted volume")
            _finish_with_exit_code(1)
        _log("Yes it does!")

        _log("Deleteing the replication controller (deletes the single nginx pod).")
        _kubectl("delete -f " + replication_controller)

        if args['destructive']:
            _log("Adding the original node back into the cluster.")
            _kubectl("uncordon " + node1)
    
    _finish_with_exit_code(0)


if __name__ == "__main__":
    _main()
