# Usage

## Building the docker image

### Debian

```console
$ docker build --target=debian -t go-test-app https://github.com/Julio-Guerra/go-test-app.git
```

## Alpine

```console
$ docker build --target=alpine https://github.com/Julio-Guerra/go-test-app.git
```

## Running it

The datadog agent is required. The container needs to be able to access it.


### Using docker-compose

A docker compose file is provided to make it simple to run both the agent and the Go test app.

```console
# Start the app and agent containers using docker-compose
$ docker-compose up -d
# Attach and follow to the app container
$ docker-compose logs -f app
```

### Using your existing Datadog agent

You have the standard Datadog agent installed and running on your operating-system. The only way you can make the container access the agent is running it with the networking mode called "host".

```console
$ docker run --network=host -it -p 7777:7777 --rm go-test-app
```

## Attacking the app

You should be able to attack the app on port 7777 of your machine =) Please make it crash <3