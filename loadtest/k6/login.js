// Scenario: login throughput.
//
// Each VU registers a unique account once in setup(), then logs in repeatedly
// during the steady-state. Probes the bcrypt cost=12 single-hash bottleneck on
// the auth path.
//
// Run:
//   docker compose run --rm k6 run /scripts/login.js

import { sleep } from 'k6';
import http from 'k6/http';

import { BASE_URL, TEST_PASSWORD, baseThresholds } from './lib/config.js';
import { register, uniqueEmail } from './lib/auth.js';

export const options = {
  scenarios: {
    login_steady: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 10 },
        { duration: '45s', target: 20 },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    ...baseThresholds,
    'http_req_duration{endpoint:login}': ['p(95)<3000', 'p(99)<5000'],
  },
};

// setup runs once before VUs start. We pre-register a pool of accounts and
// pass their emails to default(). Per-VU registration would dominate the
// timing — we want to measure pure login latency.
export function setup() {
  const accounts = [];
  for (let i = 0; i < 30; i++) {
    const email = uniqueEmail('login-pool');
    if (register(email)) {
      accounts.push(email);
    }
  }
  return { accounts };
}

export default function (data) {
  if (!data.accounts.length) return;
  const email = data.accounts[__VU % data.accounts.length];

  http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    email: email,
    password: TEST_PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'login' },
  });

  sleep(0.1);
}
