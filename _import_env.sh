#!/bin/bash

source "$(dirname "$0")"/scripts/parse-yaml.sh

echo "Importing environment variables from app.yaml file"

parse_yaml "$1"
eval "$(parse_yaml "$1")"

export PGHOST=$env_variables_POSTGRES_IP
export PGPORT=$env_variables_POSTGRES_PORT
export PGDATABASE=$env_variables_POSTGRES_DB
export PGUSER=$env_variables_POSTGRES_USER
export PGPASSWORD=$env_variables_POSTGRES_PASSWORD
export PGTESTUSER=$2

