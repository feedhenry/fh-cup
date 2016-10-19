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

## Prerequisites

- [x] Docker (for Mac* / Linux)
- [x] OpenShift 3 Client `oc`

## * Docker for Mac
- For a core, you should allocate ~6GB of memory
- You *must* add `172.30.0.0/16` as an insecure registry (via the Docker for Mac UI)
