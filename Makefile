GO     ?= go
DOCKER ?= docker
APP	   ?= dvwa

all: dvwa

dvwa:
	go mod tidy
	$(GO) build -v -tags appsec -o $(APP) ./server

clean:
	$(RM) $(APP)

test:
	$(GO) test ./...

image:
	$(DOCKER) build -t go-$(APP).

run: image
	@$(DOCKER) run -it -e APPSEC_ENABLED=1 -e DD_API_KEY="$(cat api_key.txt)" -p 8080:8080 go-$(APP)


.PHONY: all clean test image run
