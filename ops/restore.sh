#!/bin/sh
set -eu

if [ "${CONFIRM_DATABASE_RESET:-}" != "yes" ]; then
  echo "Restore overwrites application data. Set CONFIRM_DATABASE_RESET=yes." >&2
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "Usage: CONFIRM_DATABASE_RESET=yes ./ops/restore.sh backups/file.dump" >&2
  exit 1
fi

workspace="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
backup_dir="$workspace/backups"
backup_path="$(CDPATH= cd -- "$(dirname -- "$1")" && pwd)/$(basename -- "$1")"
case "$backup_path" in
  "$backup_dir"/*.dump) ;;
  *) echo "Backup must be a .dump file inside $backup_dir" >&2; exit 1 ;;
esac

name="$(basename -- "$backup_path")"
cd "$workspace"
docker compose stop backend web
docker compose exec -T postgres sh -c \
  'pg_restore --clean --if-exists --no-owner --no-privileges --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" "/backups/'"$name"'"'
docker compose up -d backend web
echo "Database restored from $backup_path"
