# Usage

## Building the docker image

### Debian

```console
$ docker build --target=debian -t go-dvwa https://github.com/DataDog/go-test-app.git
```

## Alpine

```console
$ docker build --target=alpine https://github.com/DataDog/go-test-app.git
```

## Running it

The datadog agent is required. The container needs to be able to access it.

### Using docker-compose

A docker compose file is provided to make it simple to run both the agent and
the Go test app.

```console
# Source the testing environment
$ source env.sh
# Start the app and agent containers using docker-compose
$ docker-compose up -d
# Attach and follow to the app container
$ docker-compose logs -f app
```

You can also pass custom tags with DD_TAGS and a custom service name with
DD_SERVICE.

## Attacking the app

You should be able to attack the app on port 7777 of your machine =)

For example:

1. LFI attempt:
   ```console
   curl -v --path-as-is 'http://127.0.0.1:7777/../../../etc/passwd'
   ```

2. SQLi attempt:
   ```console
   curl -v  'http://127.0.0.1:7777/sql?k=select%20*%20from%20users%20where%201%3D1%20union%20select%20*%20from%20cb'
   ```

3. SQLi vulnerability:
   ```console
   curl -v  'http://localhost:7777/products?category=%27%20union%20select%20*%20from%20user%20%27'
   ```

3. Attack attempt through the HTTP body:
   ```console
   curl -v -XPUT -d 'your json body payload' 'http://localhost:7777/api/catalog/'
   ```

Note: you can forge the ip you want by adding `-H "X-Forwarded-For: <any_ip>"` to your curl command
