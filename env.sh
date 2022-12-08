#!/bin/sh
export VAULT_ADDR=https://vault.us1.prod.dog
vault login -method=oidc
export DD_SITE=datadoghq.com

export DD_HOSTNAME=$(ddtool auth whoami --format json | jq '.name' -r)
export DD_SERVICE=peer-testing-$DD_HOSTNAME
export DD_ENV=prod-peer-testing
export DD_DOGSTATSD_NON_LOCAL_TRAFFIC="true"
export DD_REMOTE_CONFIGURATION_ENABLED="true"
export DD_API_KEY="$(vault kv get -format json applications/datadog-agent/shared/agent_api_key_appsec_test_org | jq -r .data.value)"
export DD_REMOTE_CONFIGURATION_KEY="$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_key_appsec_test_org | jq -r .data.value)"
export DD_REMOTE_CONFIGURATION_CONFIG_ROOT="$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_config_root | jq -r .data.value)"
export DD_REMOTE_CONFIGURATION_DIRECTOR_ROOT="$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_director_root | jq -r .data.value)"
export DD_REMOTE_CONFIGURATION_REFRESH_INTERVAL=5s
unset VAULT_ADDR
