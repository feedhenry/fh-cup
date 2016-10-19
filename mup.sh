# Create default MBaaS
export FH_MBAAS_OPENSHIFT_TEMPLATES=${FH_MBAAS_OPENSHIFT_TEMPLATES:-"$HOME/work/fh-openshift-templates"}
export VIRTUAL_INTERFACE_IP=192.168.44.10
export CORE_PROJECT_NAME=core
export CLUSTER_DOMAIN=cup.feedhenry.io
export MBAAS_PROJECT_NAME=mbaas1
oc new-project $MBAAS_PROJECT_NAME

installFHC() {
  if ! hash fhc 2>/dev/null; then
    echo "Installing FHC"
    npm i -g fh-fhc@latest-2
  fi
}

echo "Creating private-docker-cfg secret from ~/.docker/config.json ..."
DOCKER_CONFIG=$HOME/.docker/config.json
oc secrets new mbaas-private-docker-cfg .dockerconfigjson=$DOCKER_CONFIG
oc secrets link default mbaas-private-docker-cfg --for=pull
echo "Done."

cd $FH_MBAAS_OPENSHIFT_TEMPLATES
oc new-app -f fh-mbaas-template-1node-persistent.json

installFHC

# And link it via FHC
fhc target rhmap.cup.feedhenry.io rhmap-admin@example.com Password1
export `oc env dc/fh-mbaas --list -n mbaas1 | grep FHMBAAS_KEY`
fhc admin mbaas create --id=dev --url=https://cup.feedhenry.io:8443 --servicekey=$FHMBAAS_KEY --label=dev --username=test --password=test --type=openshift3 --routerDNSUrl="*.cup.feedhenry.io" --fhMbaasHost=https://mbaas-mbaas1.cup.feedhenry.io
sleep 30
fhc admin environments create --id=dev --label=dev --target=dev --token=`oc whoami -t`

echo "Cluster is now up: https://rhmap.cup.feedhenry.io"
echo "Login with: rhmap-admin@example.com / Password1"