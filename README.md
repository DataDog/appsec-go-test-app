# Usage

## Building the docker image

### Debian

```console
$ docker build --target=debian -t go-dvwa https://github.com/DataDog/go-test-app.git
```

## Alpine

```console
$ docker build --target=alpine https://github.com/Julio-Guerra/go-dvwa.git
```

## Running it

The datadog agent is required. The container needs to be able to access it.

### Using docker-compose

A docker compose file is provided to make it simple to run both the agent and
the Go test app.

```console
# Start the app and agent containers using docker-compose
$ env DD_API_KEY=XXX docker-compose up -d
# Attach and follow to the app container
$ docker-compose logs -f app
```

You can also pass custom tags with DD_TAGS and a custom service name with
DD_SERVICE.

### Using your existing Datadog agent

You have the standard Datadog agent installed and running on your
operating-system. The only way you can make the container access the agent is
running it with the networking mode called "host".

```console
$ docker run --network=host -it -p 7777:7777 --rm go-dvwa
```

## Attacking the app

You should be able to attack the app on port 7777 of your machine =)

For example:

1. LFI attempt:
   ```console
   curl -v --path-as-is 'http://127.0.0.1:7777/../../../etc/passwd'
   ```

3. Targeted SQLi attempt:
   ```console
   curl -v  'http://127.0.0.1:7777/sql?k=select%20*%20from%20users%20where%201%3D1%20union%20select%20*%20from%20cb'
   ```
