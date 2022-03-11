# Example of a multi-stage dockerfile creating an alpine image only containing
# the compiled Go program.

#
# Build stage compiling the test app in the golang image
#
FROM golang AS build
WORKDIR /app
COPY . .
RUN go build -v -o go-dvwa . # note that CGO is enabled by default here

#
# Final alpine stage copying the go-dvwa program compiled in the previous
# build stage into an empty alpine image, along with the required run-time
# dependencies.
#
FROM alpine
# Copy the program file from the build stage
COPY --from=build /app/go-dvwa /usr/local/bin
# Every required shared library is already present in alpine, but the C library
# doesn't have the standard name on alpine by default. Adding the libc6-compat
# package allows to add symlinks with the expected names.
RUN apk update && apk add libc6-compat
EXPOSE 7777
ENTRYPOINT [ "/usr/local/bin/go-dvwa" ]
