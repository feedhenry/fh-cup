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

- [ ] Script PV creation
- [ ] Create Core
- [ ] Call MBaaS creation script
- [ ] Link MBaaS to Core via FHC
- [ ] Capture known issues/troubleshoting

## Prerequisites

- [x] Docker (for Mac* / Linux)
- [x] OpenShift 3 Client CLI Tool `oc` version >= *[v1.3](https://github.com/openshift/origin/releases/tag/v1.3.1)*

## * Docker for Mac
- For a core, you should allocate ~6GB of memory
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)

## Troubleshooting & Known Issues

### General
-------------------

* Docker configuartion needs to be at `$HOME/.dockercfg`

### Linux Specific
-------------------

### Mac Specific
-------------------

* Running `./up.sh` produces error related to virtual interface. Solution:

```bash
export VIRTUAL_INTERFACE_IP=192.168.44.10
sudo ifconfig lo0 alias $VIRTUAL_INTERFACE_IP
```
