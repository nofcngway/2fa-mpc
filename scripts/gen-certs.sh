#!/usr/bin/env bash
#
# gen-certs.sh — generate a development PKI for mTLS between services.
#
# Layout:
#   certs/
#     ca.crt, ca.key         — root CA (sign the rest, keep ca.key offline in prod)
#     auth.{crt,key}         — auth service identity
#     twofa.{crt,key}        — twofa service identity
#     mpc-node-1.{crt,key}   — mpc node 1 identity
#     mpc-node-2.{crt,key}   — mpc node 2 identity
#     mpc-node-3.{crt,key}   — mpc node 3 identity
#     gateway.{crt,key}      — gateway client identity (no server endpoint)
#
# Each leaf cert has both serverAuth + clientAuth EKU so the same identity can
# serve and dial. SANs cover docker-compose DNS names + localhost for local runs.
#
# Usage:
#   scripts/gen-certs.sh             # idempotent: skip files that already exist
#   scripts/gen-certs.sh --force     # regenerate everything
#
# These certs are DEV-ONLY. Production must use a managed PKI (Vault, cert-manager,
# private CA) with proper rotation and key custody.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERTS_DIR="${ROOT_DIR}/certs"
FORCE=0

if [[ "${1:-}" == "--force" ]]; then
  FORCE=1
fi

mkdir -p "${CERTS_DIR}"
cd "${CERTS_DIR}"

# Skip existing artifacts unless --force.
exists() { [[ -f "$1" && $FORCE -eq 0 ]]; }

# 1. Root CA
if exists ca.key; then
  echo "skip ca.key (exists)"
else
  openssl genrsa -out ca.key 4096 2>/dev/null
  openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
    -subj "/CN=mpc-2fa-dev-ca/O=mpc-2fa/OU=dev" \
    -out ca.crt
  echo "+ generated ca.crt (10y)"
fi

# 2. Leaf certs.  service: comma-separated SAN list (DNS names).
gen_leaf() {
  local name="$1"
  local sans="$2"

  if exists "${name}.key"; then
    echo "skip ${name}.key (exists)"
    return
  fi

  # Build SAN extension. Always include "localhost" + 127.0.0.1.
  local san_ext="subjectAltName=${sans},DNS:localhost,IP:127.0.0.1"

  openssl genrsa -out "${name}.key" 2048 2>/dev/null
  openssl req -new -key "${name}.key" -subj "/CN=${name}/O=mpc-2fa/OU=dev" \
    -out "${name}.csr"

  openssl x509 -req -in "${name}.csr" -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out "${name}.crt" -days 825 -sha256 \
    -extfile <(printf "%s\nextendedKeyUsage=serverAuth,clientAuth\nbasicConstraints=critical,CA:FALSE\n" "${san_ext}")

  rm -f "${name}.csr"
  echo "+ generated ${name}.crt (825d)"
}

gen_leaf auth        "DNS:auth"
gen_leaf twofa       "DNS:twofa"
gen_leaf mpc-node-1  "DNS:mpc-node-1"
gen_leaf mpc-node-2  "DNS:mpc-node-2"
gen_leaf mpc-node-3  "DNS:mpc-node-3"
gen_leaf gateway     "DNS:gateway"

# Lock down private keys and make CA serial reproducible.
chmod 0600 *.key 2>/dev/null || true
chmod 0644 *.crt 2>/dev/null || true
rm -f ca.srl

echo
echo "PKI ready in ${CERTS_DIR}"
echo "CA fingerprint:"
openssl x509 -in ca.crt -noout -fingerprint -sha256 | sed 's/^/  /'
