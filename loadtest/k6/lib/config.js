// Shared k6 configuration. Override via environment variables when running:
//   docker compose run --rm -e BASE_URL=http://gateway:8080 k6 run /scripts/login.js

const env = (typeof __ENV !== 'undefined' ? __ENV : {});

export const BASE_URL = env.BASE_URL || 'http://gateway:8080';

// Password meets the project policy: 12+ chars, upper, lower, digit, symbol,
// no 4-char sequential runs.
export const TEST_PASSWORD = env.TEST_PASSWORD || 'L0adT3st!Secure#9';

// k6 thresholds shared across scenarios. Each scenario can extend.
export const baseThresholds = {
  http_req_failed: ['rate<0.01'],          // <1% errors
  http_req_duration: ['p(95)<2000'],       // p95 under 2s
};
