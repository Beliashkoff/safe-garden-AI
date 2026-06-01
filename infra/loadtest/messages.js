// k6 load test for POST /v1/messages (ROADMAP §6.2 DoD: 50 RPS without failures).
//
// Run against the LOCAL dev stack with the mock LLM client, so it stresses the
// handler / DB / rate-limit / SSE path without spending Anthropic tokens:
//
//   1) cd backend && LLM_CLIENT_KIND=mock make dev   # postgres+redis+minio+api
//   2) Mint dev access token(s): run the email OTP flow against the dev stack
//      (the code is shown in MailHog at http://localhost:8025), then
//   3) BASE_URL=http://localhost:8080 TOKENS=<tok1>,<tok2>,<tok3> make loadtest
//
// The per-user limit is 20 msg/s/user (ARCH §8.2). To measure ~50 RPS of real
// work rather than the limiter, spread load across at least 3 user tokens via
// the comma-separated TOKENS var. 429s are expected when a single user is
// pushed past 20/s and are NOT counted as failures below — only 5xx/timeouts
// are, which is what "without падений" means.

import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKENS = (__ENV.TOKENS || __ENV.TOKEN || '')
  .split(',')
  .map((t) => t.trim())
  .filter((t) => t.length > 0);

// Treat 200 (ok) and 429 (rate-limited, expected) as non-failures; everything
// else (5xx, timeouts) counts against http_req_failed.
http.setResponseCallback(http.expectedStatuses(200, 429));

export const options = {
  scenarios: {
    messages: {
      executor: 'constant-arrival-rate',
      rate: 50, // 50 iterations/s == 50 RPS
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'], // < 1% real errors (5xx/timeouts)
    http_req_duration: ['p(95)<5000'], // p95 < 5s, matching the latency alert
  },
};

export function setup() {
  if (TOKENS.length === 0) {
    throw new Error('set TOKEN or comma-separated TOKENS to valid access token(s)');
  }
  return { tokens: TOKENS };
}

export default function (data) {
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  const payload = JSON.stringify({
    content: [{ type: 'text', text: 'load test: what is wrong with my plant?' }],
  });
  const res = http.post(`${BASE_URL}/v1/messages`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      Accept: 'text/event-stream',
    },
  });

  check(res, {
    'status 200 or 429': (r) => r.status === 200 || r.status === 429,
    'no server error': (r) => r.status < 500,
  });
}
