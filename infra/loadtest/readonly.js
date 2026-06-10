// Read-only prod smoke load. Hits the unauthenticated /healthz through Caddy —
// no auth, no Claude calls, no DB writes. Validates the edge (Caddy + api
// liveness) under sustained read load without cost or polluting prod data.
// (messages.js is the full /v1/messages loadtest and must NOT run against prod —
// it triggers real Claude calls; use it only against a mock/staging backend.)
//
// Run from a host with decent uplink:
//   k6 run -e BASE_URL=https://api.agronomai.site infra/loadtest/readonly.js
// Tune: -e RATE=50 -e DURATION=1m
import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "https://api.agronomai.site";

export const options = {
  scenarios: {
    healthz: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 50), // requests per second
      timeUnit: "1s",
      duration: __ENV.DURATION || "1m",
      preAllocatedVUs: 20,
      maxVUs: 100,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"], // <1% failed
    http_req_duration: ["p(95)<800"], // p95 under 800ms
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/healthz`);
  check(res, { "status is 200": (r) => r.status === 200 });
}
