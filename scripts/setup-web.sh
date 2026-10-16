#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ -e .env ]; then
  printf '%s\n' '.env already exists; refusing to overwrite secrets.' >&2
  exit 1
fi
command -v openssl >/dev/null 2>&1 || { printf '%s\n' 'Install openssl first.' >&2; exit 1; }
umask 077
# noclobber also prevents overwriting an .env created after the check.
set -C
{
  printf '%s\n' 'CASTOR_IMAGE=ghcr.io/stupside/castor:latest'
  for key in CASTOR_API__TOKEN CASTOR_SERVER__TOKEN CASTOR_SCRAPING__TOKEN CASTOR_WEB_PASSWORD; do
    value=$(openssl rand -hex 32)
    printf '%s=%s\n' "$key" "$value"
  done
  printf '%s\n' 'CASTOR_WEB_USERNAME=castor' 'CASTOR_WEB_BIND=127.0.0.1' 'CASTOR_WEB_PORT=3000'
  printf '%s\n' '# REQUIRED: set your media server LAN address.' 'CASTOR_SERVER__ADVERTISE='
} > .env
printf '%s\n' 'Created .env with mode 600. Set CASTOR_SERVER__ADVERTISE before starting.'
printf '%s\n' 'See SELF_HOSTING.md for secure browser access and the first-cast checklist.'
