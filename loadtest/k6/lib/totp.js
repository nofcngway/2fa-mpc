// TOTP code generator for k6 — RFC 6238, HMAC-SHA1, 30s window, 6 digits.
//
// k6 ships with `k6/crypto.hmac` returning a hex string. We decode the base32
// secret manually (k6 only has base64 in `k6/encoding`) and assemble the
// counter as an 8-byte big-endian array.

import crypto from 'k6/crypto';

const BASE32 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';

// base32Decode converts an RFC 4648 base32 string to an Array<number> of bytes.
export function base32Decode(input) {
  const clean = input.replace(/=+$/, '').toUpperCase().replace(/\s+/g, '');
  const out = [];
  let buffer = 0;
  let bits = 0;
  for (let i = 0; i < clean.length; i++) {
    const idx = BASE32.indexOf(clean[i]);
    if (idx < 0) {
      throw new Error(`base32: invalid char ${clean[i]}`);
    }
    buffer = (buffer << 5) | idx;
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      out.push((buffer >> bits) & 0xff);
    }
  }
  return out;
}

// extractSecret pulls the base32 `secret` query param from an otpauth:// URI.
export function extractSecret(provisioningURI) {
  const m = provisioningURI.match(/[?&]secret=([A-Z2-7=]+)/i);
  if (!m) {
    throw new Error('totp: secret not found in provisioning URI');
  }
  return m[1];
}

// counterBytes encodes a uint64 counter as an 8-byte big-endian buffer.
function counterBytes(counter) {
  const bytes = new Array(8).fill(0);
  for (let i = 7; i >= 0; i--) {
    bytes[i] = counter & 0xff;
    counter = Math.floor(counter / 256);
  }
  return bytes;
}

// hmacSha1Bytes returns the HMAC-SHA1 digest as Array<number>.
function hmacSha1Bytes(keyBytes, msgBytes) {
  // k6 hmac accepts ArrayBuffer for both key and message.
  const keyAB = new Uint8Array(keyBytes).buffer;
  const msgAB = new Uint8Array(msgBytes).buffer;
  const hex = crypto.hmac('sha1', keyAB, msgAB, 'hex');
  const out = new Array(hex.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(hex.substr(i * 2, 2), 16);
  }
  return out;
}

// totp generates a 6-digit TOTP code for the given base32 secret at unixTime.
// Defaults to current time.
export function totp(base32Secret, unixTime) {
  const t = unixTime !== undefined ? unixTime : Math.floor(Date.now() / 1000);
  const counter = Math.floor(t / 30);
  const keyBytes = base32Decode(base32Secret);
  const digest = hmacSha1Bytes(keyBytes, counterBytes(counter));

  const offset = digest[digest.length - 1] & 0x0f;
  const truncated =
    ((digest[offset] & 0x7f) << 24) |
    ((digest[offset + 1] & 0xff) << 16) |
    ((digest[offset + 2] & 0xff) << 8) |
    (digest[offset + 3] & 0xff);

  const code = (truncated % 1000000).toString();
  return code.padStart(6, '0');
}
