// Scenario: realistic mixed workload (70% verify, 20% login, 10% setup).
//
// Mirrors a steady-state production traffic shape: most calls are 2FA
// verifications (every login), a smaller share is plain logins (no 2FA), and
// a small share is fresh setup (new users enabling 2FA).
//
// Run:
//   docker compose run --rm k6 run /scripts/mixed.js

import http from 'k6/http';
import { check, sleep } from 'k6';

import { BASE_URL, TEST_PASSWORD, baseThresholds } from './lib/config.js';
import { register, registerAndLogin, authHeaders, uniqueEmail } from './lib/auth.js';
import { totp, extractSecret } from './lib/totp.js';

const POOL_SIZE = 30;

export const options = {
  scenarios: {
    mixed: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 10 },
        { duration: '60s', target: 20 },
        { duration: '20s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: baseThresholds,
};

function provisionVerifyAccount() {
  const session = registerAndLogin('mixed');
  if (!session) return null;

  const setupRes = http.post(`${BASE_URL}/api/v1/2fa/setup`, JSON.stringify({
    user_id: session.userId,
    email: session.email,
  }), { headers: authHeaders(session.accessToken) });
  if (setupRes.status !== 200) return null;

  const secret = extractSecret(setupRes.json('provisioningUri'));
  const code = totp(secret);
  const enableRes = http.post(`${BASE_URL}/api/v1/2fa/verify`, JSON.stringify({
    user_id: session.userId,
    otp_code: code,
  }), { headers: authHeaders(session.accessToken) });
  if (enableRes.status !== 200 || !enableRes.json('valid')) return null;

  return { userId: session.userId, secret, accessToken: session.accessToken, email: session.email };
}

export function setup() {
  const verifyPool = [];
  const loginPool = [];

  for (let i = 0; i < POOL_SIZE; i++) {
    const acc = provisionVerifyAccount();
    if (acc) verifyPool.push(acc);
  }
  for (let i = 0; i < 10; i++) {
    const email = uniqueEmail('mixed-login');
    if (register(email)) loginPool.push(email);
  }
  return { verifyPool, loginPool };
}

function doVerify(data) {
  const acc = data.verifyPool[(__VU * 7 + __ITER) % data.verifyPool.length];
  const res = http.post(`${BASE_URL}/api/v1/2fa/verify`, JSON.stringify({
    user_id: acc.userId,
    otp_code: totp(acc.secret),
  }), { headers: authHeaders(acc.accessToken), tags: { endpoint: 'verify' } });
  check(res, { 'verify 200': (r) => r.status === 200 });
}

function doLogin(data) {
  const email = data.loginPool[__ITER % data.loginPool.length];
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: email, password: TEST_PASSWORD,
  }), { headers: { 'Content-Type': 'application/json' }, tags: { endpoint: 'login' } });
  check(res, { 'login 200': (r) => r.status === 200 });
}

function doSetup() {
  const session = registerAndLogin('mixed-setup');
  if (!session) return;
  const res = http.post(`${BASE_URL}/api/v1/2fa/setup`, JSON.stringify({
    user_id: session.userId, email: session.email,
  }), { headers: authHeaders(session.accessToken), tags: { endpoint: 'setup' } });
  check(res, { 'setup 200': (r) => r.status === 200 });
}

export default function (data) {
  if (!data.verifyPool.length || !data.loginPool.length) {
    sleep(1);
    return;
  }
  const r = Math.random();
  if (r < 0.7) doVerify(data);
  else if (r < 0.9) doLogin(data);
  else doSetup();
  sleep(2);
}
