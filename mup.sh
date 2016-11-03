# Create default MBaaS
export FH_MBAAS_OPENSHIFT_TEMPLATES=${FH_MBAAS_OPENSHIFT_TEMPLATES:-"$HOME/work/fh-mbaas-openshift-templates"}
export VIRTUAL_INTERFACE_IP=192.168.44.10
export CORE_PROJECT_NAME=core
export CLUSTER_DOMAIN=cup.feedhenry.io
export MBAAS_PROJECT_NAME=mbaas1

set -e

oc new-project $MBAAS_PROJECT_NAME

echo "Creating private-docker-cfg secret from ~/.docker/config.json ..."
DOCKER_CONFIG=$HOME/.docker/config.json
oc secrets new mbaas-private-docker-cfg .dockerconfigjson=$DOCKER_CONFIG
oc secrets link default mbaas-private-docker-cfg --for=pull
echo "Done."

cd $FH_MBAAS_OPENSHIFT_TEMPLATES
oc new-app -f fh-mbaas-template-1node-persistent.json

./link.sh
