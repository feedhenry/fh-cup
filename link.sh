installFHC() {
  if ! hash fhc 2>/dev/null; then
    echo "Installing FHC"
    npm i -g fh-fhc@latest-2
  fi
}

installFHC

# And link it via FHC
fhc target rhmap.cup.feedhenry.io rhmap-admin@example.com Password1
export `oc env dc/fh-mbaas --list -n mbaas1 | grep FHMBAAS_KEY`
fhc admin mbaas create --id=dev --url=https://cup.feedhenry.io:8443 --servicekey=$FHMBAAS_KEY --label=dev --username=test --password=test --type=openshift3 --routerDNSUrl="*.cup.feedhenry.io" --fhMbaasHost=https://mbaas-mbaas1.cup.feedhenry.io
fhc admin environments create --id=dev --label=dev --target=dev --token=`oc whoami -t`

echo "Cluster is now up: https://rhmap.cup.feedhenry.io"
echo "Login with: rhmap-admin@example.com / Password1"
