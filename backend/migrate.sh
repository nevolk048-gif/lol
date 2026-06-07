#!/bin/sh
set -e

echo "Applying migrations..."

psql $DATABASE_URL <<EOF
$(cat /app/migrations/001_init.sql)
EOF

echo "Migrations applied successfully!"
