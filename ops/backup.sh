#!/bin/sh
set -eu

name="${1:-shortli-$(date -u +%Y%m%d-%H%M%S).dump}"
case "$name" in
  */*|*.dump.dump) echo "Use a plain .dump filename" >&2; exit 1 ;;
  *.dump) ;;
  *) echo "Backup filename must end with .dump" >&2; exit 1 ;;
esac

workspace="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
mkdir -p "$workspace/backups"
cd "$workspace"
docker compose exec -T postgres sh -c \
  'pg_dump --format=custom --no-owner --no-privileges --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" --file="/backups/'"$name"'"'
echo "Backup created: $workspace/backups/$name"
