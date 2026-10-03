#!/bin/bash
set -e

# ждём Postgres — healthcheck на compose делает то же, но перестраховка
until pg_isready -h postgres -U postgres; do
  echo "waiting for postgres..."
  sleep 1
done

# порядок миграций = алфавитный внутри папки. Для каждой БД — свой набор.
for svc in users catalog cart orders payments notifications; do
  echo "=== migrations for $svc ==="
  for f in /migrations/$svc/*.sql; do
    echo "applying $f"
    PGPASSWORD=postgres psql -v ON_ERROR_STOP=1 \
      -h postgres -U postgres -d "$svc" -f "$f"
  done
done

echo "All migrations applied."
