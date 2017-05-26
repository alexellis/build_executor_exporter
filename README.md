# build_executor_exporter

Prometheus exporter for Jenkins Build executors metrics. Use this with AlertManager to build alerting for your remote executors/agents/slaves.

* `online_status`

Whether a node has become disconnected

* `temporarily_offline_status`

Whether a node is deliberately marked as offline for maintenance 

### Options for running:

The default port for the exporter is TCP/9001 and the metrics endpoint is /metrics.

* As a long running daemon inside a Docker container

```
docker run -p 9001:9001 -d alexellis2/build_executor_exporter:0.3-faas ./build_executor_exporter -urls http://site1,http://site2
```

* As a native Golang binary

Run `go install` and use with `-urls http://site1,http://site2`

* As a serverless / one-shot Docker image:

This wil run once and then output to `stdout` and can be used with [FaaS](https://github.com/alexellis/faas) as a serverless function.

```
docker run -ti alexellis2/build_executor_exporter:0.3-faas ./build_executor_exporter -urls http://site1,http://site2
```

#### Sample alert.rules

This is an example of alerting rules which can be configured with AlertManager:

```
ALERT AgentOffline 
  IF online_status == 0 
  FOR 60s
  LABELS {
    service = "build_executor_exporter",
    severity = "major",
    value = "{{$value}}"
  }
  ANNOTATIONS {
    summary = "Agent down on {{ $labels.instance }}",
    description =  "High invocation total on {{ $labels.instance }} {{ $value }}"
  } 
```

You may want to combine this with the temporarily_offline_status flag if you have build agents which are kept offline for long periods of maintenance.

### Basic auth

If your Jenkins server has security enabled then you can use Docker Swarm secrets to securely encrypt and share the credentials. Otherwise if running outside of Swarm you can mount a folder with the password by setting the `secret_store` environmental variable.

Example:

```
docker secret create username -
```
Secret is read from stdin followed by hitting `Control + D`

Specify the username (only) in the `-urls` flag such as:

```
./build_executor_exporter -urls https://username@host:8080`
```

When running in swarm the secret will be available under `/run/secrets/username`.

## License

MIT
