// Scenario: 2FA Setup throughput.
//
// Each iteration: register → login → POST /api/v1/2fa/setup. Setup is the
// heaviest endpoint — it does TOTP secret generation, Shamir split, parallel
// gRPC fan-out to 3 MPC nodes, AES encryption per share, and parallel bcrypt
// of 10 backup codes (Phase A optimization). Probes the parallel-bcrypt path
// and MPC distribution throughput.
//
// Run:
//   docker compose run --rm k6 run /scripts/setup-2fa.js

import http from 'k6/http';
import { check, sleep } from 'k6';

import { BASE_URL, baseThresholds } from './lib/config.js';
import { registerAndLogin, authHeaders } from './lib/auth.js';

export const options = {
  scenarios: {
    setup_steady: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 5 },
        { duration: '45s', target: 10 },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    ...baseThresholds,
    'http_req_duration{endpoint:setup}': ['p(95)<3000', 'p(99)<5000'],
  },
};

export default function () {
  const session = registerAndLogin('setup');
  if (!session) {
    sleep(0.5);
    return;
  }

  const res = http.post(`${BASE_URL}/api/v1/2fa/setup`, JSON.stringify({
    user_id: session.userId,
    email: session.email,
  }), {
    headers: authHeaders(session.accessToken),
    tags: { endpoint: 'setup' },
  });

  check(res, {
    'setup 200': (r) => r.status === 200,
    'has provisioning_uri': (r) => r.status === 200 && r.json('provisioningUri').includes('otpauth://'),
    'has 10 backup codes': (r) => r.status === 200 && r.json('backupCodes').length === 10,
  });

  sleep(0.5);
}
