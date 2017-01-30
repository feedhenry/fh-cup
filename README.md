# fh-cup
```
 ( (
  ) )
........
|  fh  |]
\      /
 `----'
```

Wrapper CLI for `oc cluster up` to give you a working local RHMAP core for development with OpenShift Origin.

# Installation and Running

* Clone this repo to your $GOPATH
* Copy `fh-cup.toml` to `~/.fh-cup.toml` and configure the paths and Docker Hub details
# Install dependencies via `glide i` using Glide
* Build `fh-cup` via `go build`
* Run `./fh-cup up --clean`

Some other commands and options

* `./fh-cup down --clean # Tear down cluster & clean leftover config/persistence`
* `./fh-cup check # Check for pre-requisites (WIP)`
* `./fh-cup up --skip-image-seeding # Skip image seeding (Not recommended)` 

## TODO
- [ ] Drop in config into `~/.fh-cup.toml` (maybe interactive)
- [ ] More pre-flight checks

## Prerequisites

- [x] Docker (for Mac* / Linux**)
- [x] OpenShift 3 Client CLI Tool `oc` version >= *[v1.3](https://github.com/openshift/origin/releases/tag/v1.3.1)*
- [x] `socat` installed
- [x] `docker` logged in to a Docker Hub account with access to the rhmap project

## * Docker for Mac
- For a core, you should allocate ~7GB of memory
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)

## ** Docker for Linux

[CF: https://github.com/openshift/origin/blob/master/docs/cluster_up_down.md#getting-started]

| WARNING |
| ------- |
| The default Firewalld configuration on Fedora blocks access to ports needed by containers running on an OpenShift cluster. Make sure you grant access to these ports. See step 3 below. |
| Check that `sysctl net.ipv4.ip_forward` is set to 1. |

1. Install Docker with your platform's package manager.
2. Configure the Docker daemon with an insecure registry parameter of `172.30.0.0/16`
   - In RHEL and Fedora, edit the `/etc/sysconfig/docker` file and add or uncomment the following line:
     ```
     INSECURE_REGISTRY='--insecure-registry 172.30.0.0/16'
     ```

   - After editing the config, restart the Docker daemon.
     ```
     $ sudo systemctl restart docker
     ```
3. Ensure that your firewall allows containers access to the OpenShift master API (8443/tcp) and DNS (53/udp) endpoints.
   In RHEL and Fedora, you can create a new firewalld zone to enable this access:
   - Determine the Docker bridge network container subnet:
     ```
     docker network inspect bridge -f "{{range .IPAM.Config }}{{ .Subnet }}{{end}}"
     ```
     You will should get a subnet like: ```172.17.0.0/16```

   - Create a new firewalld zone for the subnet and grant it access to the API and DNS ports:
     ```
     firewall-cmd --permanent --new-zone dockerc
     firewall-cmd --permanent --zone dockerc --add-source 172.17.0.0/16
     firewall-cmd --permanent --zone dockerc --add-port 8443/tcp
     firewall-cmd --permanent --zone dockerc --add-port 53/udp
     firewall-cmd --reload
     ```

## Troubleshooting & Known Issues

### General
-------------------

* Docker configuration needs to be at `$HOME/.docker/config.json` - login via `docker login`

### FHC Core & MBaaS linking

If you see an error similar to the following:

```
fhc ERR! Error: getaddrinfo ENOTFOUND rhmap.cup.feedhenry.io rhmap.cup.feedhenry.io:443
```

This is likely because this hostname is not resolved correctly. `*.cup.feedhenry.io` should resolve to `192.168.44.10`. Verify this with `dig` or similar, for example:

```
dig rhmap.cup.feedhenry.io
```
