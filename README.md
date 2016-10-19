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
- [x] OpenShift 3 Client `oc`

## * Docker for Mac
- For a core, you should allocate ~6GB of memory
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)
