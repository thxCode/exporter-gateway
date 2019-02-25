# Rancher Monitoring Exporter Gateway

`exporter-gateway` is like a gateway to aggregate the local exporters and push all metrics to the [pushgateway](https://github.com/prometheus/pushgateway).

## How to use

### Running parameters

```bash
NAME:
   exporter-gateway - Gateway for the Prometheus exporters

USAGE:
   exporter-gateway [global options] command [command options] [arguments...]

VERSION:
   ...

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --interval value      [optional] Set the transmission interval (default: 15s)
   --from value          Set the exporter routers, for example, transferring from localhost xyz exporter by '--from xyz-exporter.url=https://127.0.0.1:9010/metrics' (default: [])
   --from.timeout value  [optional] Set timeout for scraping, must less than --interval (default: 10s)
   --to value            Set the pushgateway routers, for example, transferring to x.y.z pushgateway by '--to xyz-pushgateway.url=http://x.y.z:9091/metrics' (default: [])
   --to.group value      [optional] Set group for the metrics, for example, grouping all metrics by '--to.group instance=l.m.n'
   --log.json            [optional] Log as JSON
   --log.debug           [optional] Log debug info
   --help, -h            show help
   --version, -v         print the version
```

#### Router configuration for `--from`, `--to`
```json
{
  "url": "", // Set URL for HTTP endpoint.
  "bearerToken": "", // Set the `Authorization` header on every HTTP request with the configured bearer token. It is mutually exclusive with `bearerTokenFile`.
  "bearerTokenFile": "<filename>", // Set the `Authorization` header on every HTTP request with the bearer token read from the configured file. It is mutually exclusive with `bearerToken`.
  "basicAuth": { // Set the `Authorization` header on every HTTP request with the configured username and password. `password` and `passwordFile` are mutually exclusive.
    "username": "",
    "password": "",
    "passwordFile": "<filename>"
  },
  "tlsConfig": { // Set the HTTP request's TLS settings.
    "caFile": "<filename>", // CA certificate to validate the HTTP endpoint server certificate.
    "certFile": "<filename>", // Certificate file for client cert authentication to the HTTP endpoint server.
    "keyFile": "<filename>", // Key file for client cert authentication to the HTTP endpoint server.
    "serverName": "", // ServerName extension to indicate the name of the HTTP endpoint server.
    "insecureSkipVerify": "" // Disable validation of the HTTP endpoint server certificate.
  }
}
```

### Start example

```bash
exporter-gateway \
    --interval 30s
    --from kube-controller-manager.url=http://127.0.0.1:10252/metrics \
    --from kube-scheduler.url=http://127.0.0.1:10251/metrics \
    --from etcd.url=http://127.0.0.1:2379/metrics \
    --from etcd.tlsConfig.caFile=/var/etcd/ca.crt \
    --from etcd.tlsConfig.certFile=/var/etcd/peer.crt \
    --from etcd.tlsConfig.keyFile=/var/etcd/peer.key \
    --to ps1.url=http://pushgateway.cattle-prometheus.svc.cluster.local \
    --to.group instance=master.node.ip
```

# License

Copyright (c) 2014-2019 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.