# Example of a multi-stage dockerfile creating an alpine image only containing
# the compiled Go program.

#
# Build stage compiling the test app in the golang image
#
FROM golang AS build

WORKDIR /app
COPY . .

# 1. Compile the program
RUN go build -v -o go-test-app . # note that CGO is enabled by default here

# 2. Prepare a directory with the shared libraries required by the compiled
# program file using ldd:
# 2.1. Install ldd from the binutils pacakge
RUN apt update && apt install -y binutils
# 2.2. Use ldd to list the shared libraries and copy them into libs/
RUN ldd go-test-app | tr -s '[:blank:]' '\n' | grep '^/' | \
      xargs -I % sh -c 'mkdir -p $(dirname libs%); cp % libs%;'

#
# Final stage copying the go-test-app program compiled in the previous build
# stage into an empty scratch image, along with its required run-time
# dependencies.
#
FROM scratch
# Copy the program file from the build stage along with the shared libraries
# it requires.
COPY --from=build /app/libs /app/go-test-app /
EXPOSE 7777
ENTRYPOINT [ "/go-test-app" ]