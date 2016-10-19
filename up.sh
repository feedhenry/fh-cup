#!/usr/bin/env bash

set -e
#set -x

SCRIPT_DIR="$( cd $( dirname "${BASH_SOURCE[0]}" ) && pwd)"
CLUSTER_DIR="$SCRIPT_DIR/cluster"
VIRTUAL_INTERFACE_IP=192.168.44.10
FH_CORE_OPENSHIFT_TEMPLATES="$HOME/work/fh-core-openshift-templates"
BASE_PV_DIR="/tmp"
export CORE_PROJECT_NAME=core
export CLUSTER_DOMAIN=$VIRTUAL_INTERFACE_IP.xip.io

FLUSH_IPTABLES=${FLUSH_IPTABLES:-"false"}

echo "Checking pre-requisities..."
echo "Done."

if [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  # Linux

  # turn on promiscuous mode for docker interface
  # Need for 'hairpinning' issues from Pods back to Services
  echo "Enabling promiscuous mode for docker0 - may be prompted for password"
  sudo ip link set docker0 promisc on

  # If this workaround is enabled, flush ip tables
  # This works around dns issues in containers e.g. 'cannot clone from github.com' when doing an s2i build
  if [ "$FLUSH_IPTABLES" == "true" ]; then
    echo "Flushing iptables"
    sudo iptables-save > $CLUSTER_DIR/iptables.backup.$(date +"%s")
    sudo iptables -F
  fi
fi

# Setup Virtual interface for our cluster, so the cluster's
# IP does not shift when switching networks (e.g. wired => wifi)
function setupInterface {
  if [ "$(uname)" == "Darwin" ]; then
    # macOS
    sudo ifconfig lo0 alias $VIRTUAL_INTERFACE_IP
  elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    sudo ifconfig lo:0 $VIRTUAL_INTERFACE_IP
  fi
}

# Destroy previous virtual interface
function destroyInterface {
  if [ "$(uname)" == "Darwin" ]; then
    # macOS
    sudo ifconfig lo0 -alias $VIRTUAL_INTERFACE_IP
  elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    set +e
    LO_VIRTUAL_INTERFACE=$(ifconfig | grep "lo:0")
    set -e
    if [ "$LO_VIRTUAL_INTERFACE" ]; then
      echo "Removing virutal interface for $VIRTUAL_INTERFACE_IP"
      sudo ifconfig lo:0 down
    fi
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

echo "Creating cluster directories if they do not exist..."
mkdir -p $CLUSTER_DIR/data $CLUSTER_DIR/config $CLUSTER_DIR/volumes

echo "Running 'oc cluster up'..."

oc cluster up --host-data-dir=$CLUSTER_DIR/data --host-config-dir=$CLUSTER_DIR/config --public-hostname=$VIRTUAL_INTERFACE_IP --routing-suffix=$VIRTUAL_INTERFACE_IP.xip.io
# TODO: Check !=0 return
echo "Cluster up, continuing."

echo "Creating PVs..."
asSysAdmin
sleep 1
# we do a global find and replace as the list will contain more than one pv
sed -i -e "s|{BASE_PLACEHOLDER_DIR}|${BASE_PV_DIR}|g" pvs-template.yml
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

cd $SCRIPT_DIR
