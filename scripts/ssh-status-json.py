#!/usr/bin/python3
# (c) Bernhard Tittelbach, 2019

### this file replaces ssh-status-json.sh
### meant to be run via .ssh/authorized_keys as forced command=

import json
import sys
import os
import shutil

env_ssh_original_command_ = "SSH_ORIGINAL_COMMAND"
spaceapi_status_dst_filepath_="/dev/shm/www/spaceapi.json"
spaceapi_status_tmp_filepath_="/dev/shm/www/spaceapi.json.tmp"

def ensureParentDirExists(filename):
    dn = os.path.dirname(spaceapi_status_tmp_filepath_)
    if not os.path.exists(dn):
        os.makedirs(dn)

def cmd_get():
    try:
        with open(spaceapi_status_dst_filepath_,"r") as fh:
            print(fh.read())
    except Exception as e:
        print(e)
        sys.exit(0)

def cmd_set():
    spacestatus=""
    try:
        spacestatus = json.load(sys.stdin)
    except Exception as e:
        print("ERROR:",e)
        sys.exit(0)

    ensureParentDirExists(spaceapi_status_tmp_filepath_)
    with open(spaceapi_status_tmp_filepath_,"w") as fh:
        fh.write(json.dumps(spacestatus))

    ensureParentDirExists(spaceapi_status_dst_filepath_)
    shutil.move(spaceapi_status_tmp_filepath_,spaceapi_status_dst_filepath_)


def helpexit():
    print("Valid commands: get | set")
    sys.exit(1)

def main():
    if not env_ssh_original_command_ in os.environ:
        helpexit()

    command = os.environ[env_ssh_original_command_].split(" ")[0]
    commandset_dict = {"get":cmd_get,"set":cmd_set}

    if command in commandset_dict:
        commandset_dict[command]()
    else:
        helpexit()


if __name__ == '__main__':
    main()

