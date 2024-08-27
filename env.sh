#!/bin/sh
# check for --staging  or -s argument
if [ "$1" = "--staging" ] || [ "$1" = "-s" ]; then
    export VAULT_ADDR=https://vault.us1.staging.dog
    vault login -method=oidc
    export DD_SITE=datad0g.com
    export DD_API_KEY=$(vault kv get -format json applications/datadog-agent/shared/agent_api_key_automated_blocking_test | jq -r .data.value)
    export DD_REMOTE_CONFIGURATION_CONFIG_ROOT=$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_config_root_automated_blocking_test | jq -r .data.value)
    export DD_REMOTE_CONFIGURATION_DIRECTOR_ROOT=$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_director_root_automated_blocking_test | jq -r .data.value)
    export DD_ENV=staging
else
    export VAULT_ADDR=https://vault.us1.prod.dog
    vault login -method=oidc
    export DD_SITE=datadoghq.com
    export DD_ENV=prod-peer-testing
    export DD_API_KEY="$(vault kv get -format json applications/datadog-agent/shared/agent_api_key_appsec_test_org | jq -r .data.value)"
    export DD_REMOTE_CONFIGURATION_KEY="$(vault kv get -format json applications/datadog-agent/shared/agent_remote_config_key_appsec_test_org | jq -r .data.value)"
fi
export DD_REMOTE_CONFIGURATION_ENABLED=true
export DD_HOSTNAME=$(ddtool auth whoami --format json | jq '.name' -r)
export DD_SERVICE=solo-testing-$DD_HOSTNAME
export DD_DOGSTATSD_NON_LOCAL_TRAFFIC="true"
export DD_REMOTE_CONFIGURATION_ENABLED="true"
unset VAULT_ADDR
