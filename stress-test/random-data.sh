#!/bin/bash

data=$(head -n $RANDOM /dev/urandom | tr -dc '[:alpha:]')
echo "{ \"random-json-data\": \"$data\" }" &>body.json
