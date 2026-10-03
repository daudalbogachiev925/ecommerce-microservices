#!/bin/bash
set -e

# Создаём 5 баз — по одной на сервис (database-per-service)
for db in users catalog cart orders payments notifications; do
  echo "Creating database: $db"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE $db;
    GRANT ALL PRIVILEGES ON DATABASE $db TO $POSTGRES_USER;
EOSQL
done

echo "All databases created."
