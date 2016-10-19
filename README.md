# fh-cup

```
 ( (
  ) )
........
|      |]
\      /
 `----'
```

Some scripts to wrap `oc cluster up` to give you a working local RHMAP core for development with OpenShift Origin.

## TODO

- [x] Script PV creation
- [x] Create Core
- [ ] Call MBaaS creation script
- [ ] Link MBaaS to Core via FHC

## Prerequisites

- [x] Docker (for Mac* / Linux**)
- [x] OpenShift 3 Client CLI Tool `oc` version >= *[v1.3](https://github.com/openshift/origin/releases/tag/v1.3.1)*
- [x] `socat` installed
- [x] `docker` logged in to a Docker Hub account with access to the rhmap project

## * Docker for Mac
- For a core, you should allocate ~6GB of memory
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)

## Troubleshooting & Known Issues

### General
-------------------

* Docker configuration needs to be at `$HOME/.docker/config.json` - login via `docker login`

### Linux Specific
-------------------

#### ** RHEL & Fedora

* Running `./up.sh` produces error related to virtual interface. Solution:
* Add the following to `/etc/sysconfig/docker`
```bash
DOCKER_OPTS="--insecure-registry 172.30.0.0/16"
```

### Mac Specific
-------------------

* Running `./up.sh` produces error related to virtual interface. Solution:

```bash
export VIRTUAL_INTERFACE_IP=192.168.44.10
sudo ifconfig lo0 alias $VIRTUAL_INTERFACE_IP
```
