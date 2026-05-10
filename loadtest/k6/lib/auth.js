// Auth helpers for k6 scenarios — registration, login, header construction.

import http from 'k6/http';
import { check } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

import { BASE_URL, TEST_PASSWORD } from './config.js';

// uniqueEmail builds a collision-resistant email. __VU and __ITER are only
// defined inside VU iterations, so fall back to randomness when called from
// setup() / teardown() (k6 init context).
export function uniqueEmail(prefix) {
  const vu = (typeof __VU !== 'undefined' && __VU > 0) ? __VU : 0;
  const iter = (typeof __ITER !== 'undefined') ? __ITER : 0;
  return `${prefix || 'load'}-${vu}-${iter}-${randomString(8)}@loadtest.local`;
}

// register hits POST /api/v1/auth/register, returns {tokens, user}.
export function register(email) {
  const res = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
    email: email,
    password: TEST_PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'register' },
  });
  check(res, { 'register 200': (r) => r.status === 200 });
  if (res.status !== 200) {
    return null;
  }
  return res.json();
}

// login hits POST /api/v1/auth/login, returns {tokens, user}.
export function login(email) {
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: email,
    password: TEST_PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'login' },
  });
  check(res, { 'login 200': (r) => r.status === 200 });
  if (res.status !== 200) {
    return null;
  }
  return res.json();
}

// authHeaders builds the Authorization header for a given access token.
export function authHeaders(accessToken) {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${accessToken}`,
  };
}

// registerAndLogin is a single-iteration helper: creates a user and logs in.
// Returns {accessToken, refreshToken, userId, email} or null on failure.
export function registerAndLogin(prefix) {
  const email = uniqueEmail(prefix);
  const reg = register(email);
  if (!reg) return null;
  // grpc-gateway emits camelCase JSON keys regardless of proto snake_case.
  return {
    accessToken: reg.tokens.accessToken,
    refreshToken: reg.tokens.refreshToken,
    userId: reg.user.id,
    email: email,
  };
}
