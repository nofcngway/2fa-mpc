// Scenario: 2FA Verify throughput.
//
// setup() pre-provisions a pool of accounts with 2FA enabled (register →
// login → setup → first verify to flip is_enabled=true). The default()
// function then loops: pick an account → compute current TOTP code →
// POST /api/v1/2fa/verify. Probes:
//   - 2-of-3 Shamir share retrieval from MPC nodes
//   - mTLS handshake overhead (connection reused, so warm path)
//   - OTP-reuse Redis check
//
// OTP reuse is non-trivial: the service rejects the same counter twice, so
// we space iterations apart with sleep(31s) ÷ accounts so each account sees
// a fresh window. Pool size = enough accounts to keep VUs busy without reuse.
//
// Run:
//   docker compose run --rm k6 run /scripts/verify-2fa.js

import http from 'k6/http';
import { check, sleep } from 'k6';

import { BASE_URL, baseThresholds } from './lib/config.js';
import { registerAndLogin, authHeaders } from './lib/auth.js';
import { totp, extractSecret } from './lib/totp.js';

// Pool sized to keep each account well under TwoFA's 5-verifies-per-5-minutes
// rate limit — at 5 VUs over 90s with 5s spacing, each account is touched
// roughly once. Larger VU counts demand a proportionally larger pool.
const POOL_SIZE = 80;

export const options = {
  scenarios: {
    verify_steady: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 3 },
        { duration: '60s', target: 5 },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    ...baseThresholds,
    'http_req_duration{endpoint:verify}': ['p(95)<1000', 'p(99)<2000'],
  },
};

// Provision an account with 2FA enabled. Returns {userId, secret, accessToken}.
function provisionAccount() {
  const session = registerAndLogin('verify');
  if (!session) return null;

  const setupRes = http.post(`${BASE_URL}/api/v1/2fa/setup`, JSON.stringify({
    user_id: session.userId,
    email: session.email,
  }), { headers: authHeaders(session.accessToken) });

  if (setupRes.status !== 200) return null;
  const secret = extractSecret(setupRes.json('provisioningUri'));

  // First verify enables 2FA. Use current TOTP.
  const code = totp(secret);
  const verifyRes = http.post(`${BASE_URL}/api/v1/2fa/verify`, JSON.stringify({
    user_id: session.userId,
    otp_code: code,
  }), { headers: authHeaders(session.accessToken) });

  if (verifyRes.status !== 200 || !verifyRes.json('valid')) return null;

  return {
    userId: session.userId,
    secret: secret,
    accessToken: session.accessToken,
  };
}

export function setup() {
  const accounts = [];
  for (let i = 0; i < POOL_SIZE; i++) {
    const acc = provisionAccount();
    if (acc) accounts.push(acc);
  }
  return { accounts };
}

export default function (data) {
  if (!data.accounts.length) return;
  // Spread VUs across the pool; iter offset prevents two VUs hitting the same
  // account in lockstep.
  const idx = (__VU * 7 + __ITER) % data.accounts.length;
  const acc = data.accounts[idx];
  const code = totp(acc.secret);

  const res = http.post(`${BASE_URL}/api/v1/2fa/verify`, JSON.stringify({
    user_id: acc.userId,
    otp_code: code,
  }), {
    headers: authHeaders(acc.accessToken),
    tags: { endpoint: 'verify' },
  });

  check(res, {
    'verify 200': (r) => r.status === 200,
  });

  // 5s spacing × 5 VUs ≈ 1 verify/s/account on a pool of 80 — well below
  // the TwoFA rate limit (5/5min/user) and the 30s TOTP-reuse window.
  sleep(5);
}
