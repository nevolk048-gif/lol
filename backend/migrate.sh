#!/bin/sh
set -e

echo "Applying migrations..."

for migration in /app/migrations/*.sql; do
  echo "Running migration: $(basename $migration)"
  psql $DATABASE_URL < "$migration"
done

echo "All migrations applied successfully!"
