# Example of a multi-stage dockerfile creating an arm64 debian image with a Go
# program cross-compiled from the golang image.

#
# Build stage compiling the app in the golang image
#
FROM golang AS build

# Install gcc for arm64
RUN apt-get update && apt-get install -y gcc-aarch64-linux-gnu

WORKDIR /app
COPY . .

# Cross-compile the Go program for linux/arm64
RUN CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build -v -tags appsec -o go-dvwa .

#
# Final debian arm64 stage copying the go-dvwa program compiled in the previous
# build stage into the empty debian image.
#
FROM arm64v8/debian
# Copy the program file from the build stage
COPY --from=build /app/go-dvwa /usr/local/bin
EXPOSE 7777
ENTRYPOINT [ "/usr/local/bin/go-dvwa" ]