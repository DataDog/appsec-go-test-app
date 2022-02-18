# Docker image examples

The sub-folders you will find here contain Dockerfile examples creating docker
images for commonly found distributions.

They show-case multi-stage docker builds which allow to separate the build
environment from the final production application image.

They all use the official [golang docker image](https://hub.docker.com/_/golang)
which already contains everything required to compile a Go program with AppSec.
