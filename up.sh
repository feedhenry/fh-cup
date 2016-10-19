#!/usr/bin/env bash

VIRTUAL_INTERFACE_IP=192.168.44.10
FH_CORE_OPENSHIFT_TEMPLATES="/Users/jasonmadigan/Work/fh-core-openshift-templates"
PV_DIR="/Users/jasonmadigan/Work/fh-cup/cluster"
export CORE_PROJECT_NAME=core
export CLUSTER_DOMAIN=$VIRTUAL_INTERFACE_IP.xip.io
FH_CUP=`pwd`

echo "Checking pre-requisities..."
echo "Done."

# Setup Virtual interface for our cluster, so the cluster's
# IP does not shift when switching networks (e.g. wired => wifi)
function setupInterface {
  if [ "$(uname)" == "Darwin" ]; then
    # macOS
    sudo ifconfig lo0 alias $VIRTUAL_INTERFACE_IP
  elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    ifconfig lo:0 $VIRTUAL_INTERFACE_IP
  fi
}

# Destroy previous virtual interface
function destroyInterface {
  if [ "$(uname)" == "Darwin" ]; then
    # macOS
    sudo ifconfig lo0 -alias $VIRTUAL_INTERFACE_IP
  elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    echo "TODO"
  fi
}

function asDeveloper {
  oc login -u developer -p developer
}

function asSysAdmin {
  echo "Switching to system:admin in oc"
  oc login -u system:admin
  echo "Done."
}

echo "Setting up Virtual Interace for oc cluster with IP: $VIRTUAL_INTERFACE_IP"
echo "Removing previous interface(s) - may be prompted for password"
destroyInterface
echo "Done. Creating new interface..."
setupInterface
echo "Done."

echo "Creating PV directories if they do not exist..."
mkdir -p $PV_DIR/data $PV_DIR/config $PV_DIR/volumes

echo "Running 'oc cluster up'..."
oc cluster up --host-data-dir=$PV_DIR/data --host-config-dir=$PV_DIR/config --public-hostname=$VIRTUAL_INTERFACE_IP --routing-suffix=$CLUSTER_DOMAIN
# TODO: Check !=0 return
echo "Cluster up, continuing."

echo "Creating PVs..."
asSysAdmin
sleep 1
oc create -f ./pvs-template.yml
echo "Done."

echo "Creating Core Project..."
asDeveloper
oc new-project $CORE_PROJECT_NAME
echo "Done."

echo "Running Core setup scripts...."

cd $FH_CORE_OPENSHIFT_TEMPLATES/scripts/core
echo "Running prerequisites.sh..."
./prerequisites.sh
asSysAdmin
oc create -f $FH_CORE_OPENSHIFT_TEMPLATES/gitlab-shell/scc-anyuid-with-chroot.json
oc adm policy add-scc-to-user anyuid-with-chroot system:serviceaccount:${CORE_PROJECT_NAME}:default
asDeveloper
echo "Done."

echo "Running infra setup..."
./infra.sh
echo "Done."

cd $FH_CUP