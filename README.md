# `tempest_exporter` [![source](https://img.shields.io/badge/source-gray?logo=github)](https://github.com/willglynn/tempest_exporter)

This is a Prometheus/OpenMetrics exporter for [Tempest weather
stations](https://weatherflow.com/tempest-home-weather-system/).

This tool listens for [Tempest UDP broadcasts](https://weatherflow.github.io/Tempest/api/udp.html) and forwards metrics
to a Prometheus push gateway.

## Quickstart

Container images are available at [Docker Hub](https://hub.docker.com/r/willglynn/tempest_exporter) and [GitHub
container registry](https://github.com/willglynn/tempest_exporter/pkgs/container/tempest_exporter).

```shell
$ docker run -it --rm --net=host \
  -e PUSH_URL=http://victoriametrics:8429/api/v1/import/prometheus \
  willglynn/tempest_exporter
# or
$ docker run -it --rm --net=host \
  -e PUSH_URL=http://victoriametrics:8429/api/v1/import/prometheus \
  ghcr.io/willglynn/tempest_exporter
2023/07/06 20:18:55 pushing to "0.0.0.0" with job name "tempest"
2023/07/06 20:18:55 listening on UDP :50222
```

Note that `--net=host` is used here because UDP broadcasts are link-local and therefore cannot be received from typical
(routed) container networks.

## Exporter configuration

Minimal, via environment variables:

* `PUSH_URL`: the URL of the [Prometheus pushgateway](https://github.com/prometheus/pushgateway) or other [compatible
  service](https://docs.victoriametrics.com/?highlight=exposition#how-to-import-data-in-prometheus-exposition-format)
* `JOB_NAME`: the value for the `job` label, defaulting to `"tempest"`

## Status

This works for me and my Tempest setup. Feel free to open pull requests with proposed changes.
