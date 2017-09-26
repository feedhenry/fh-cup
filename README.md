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

# CI
PRs are built using Travis CI

# Installation and Running

* Clone this repo to your $GOPATH
  * You'll also need to clone [rhmap-ansible](https://github.com/fheng/rhmap-ansible), [fh-core-openshift-templates](https://github.com/fheng/fh-core-openshift-templates) & [fh-openshift-templates](https://github.com/feedhenry/fh-openshift-templates) to a working directory
* Copy `fh-cup.toml` to `~/.fh-cup.toml` and configure the paths and Docker Hub details
# Install dependencies via `glide i` using Glide
* Build `fh-cup` via `go build`
* Run `./fh-cup up --clean`

Some other commands and options

* `./fh-cup down --clean # Tear down cluster & clean leftover config/persistence`
* `./fh-cup check # Check for pre-requisites (WIP)`
* `./fh-cup up --skip-image-seeding # Skip image seeding (Not recommended)` 
* `./fh-cup install # Run rhmap-ansible installer on an already running cluster`
* `./fh-cup seed # Seed RHMAP Core & MBaaS images into Docker`

## TODO
- [ ] Drop in config into `~/.fh-cup.toml` (maybe interactive)
- [ ] More pre-flight checks
- [ ] Embed templates into binary - use embedded by default. Single distributable.
- [ ] Build for multiple platforms

## Prerequisites

- [x] Docker (for Mac* / Linux**)
- [x] OpenShift 3 Client CLI Tool `oc` version >= *[v1.3](https://github.com/openshift/origin/releases/tag/v1.3.1)*
- [x] `socat` installed
- [x] `docker` logged in to a Docker Hub account with access to the rhmap project

## * Docker for Mac
- For an RHMAP Core & MBaaS install, you should allocate ~7GB of memory & > 5 CPU cores
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)
- Kubernetes is currently broken when using CE editions of Docker. For macOS, it's recommended to use [Docker 1.13.1](https://download.docker.com/mac/stable/1.13.1.15353/Docker.dmg) for now.

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
     firewall-cmd --permanent --zone dockerc --add-port 443/tcp
     firewall-cmd --permanent --zone dockerc --add-port 80/tcp
     firewall-cmd --reload
     ```

## Troubleshooting & Known Issues

#### General

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

#### `No such file or folder` when looking for templates/generated/fh-core-XXXXX.json

* Ensure the `generated/` folder is populated, if not, run `npm install` and `grunt` from the templates root
* Make sure to use complete paths in `~/.fh-cup.toml` eg. `/Users/ecrosbie/dir` instead of `~/dir`

### Error: Waiting for API server to start listening

If you see the following error (usually on a Mac):

```
   Waiting for API server to start listening
FAIL
   Error: timed out waiting for OpenShift container "origin"
   WARNING: 192.168.44.10:8443 may be blocked by firewall rules
   Details:
     No log available from "origin" container
```

Along with lots of lines in the origin container logs with the following:

```
I0926 20:55:00.077851   29405 logs.go:41] http: TLS handshake error from 127.0.0.1:37680: EOF
```

It's likely that traffic is being blocked by a firewall or filter (e.g. Little Snitch). Disable or allow access to fix.