import grpc from "k6/net/grpc";
import { check, fail } from "k6";
import { Rate } from "k6/metrics";

// ---------- ENV PARAMS ----------
const TARGET = __ENV.TARGET || "localhost:50051"; // host:port gRPC
const RATE = Number(__ENV.RATE || "1000"); // requests per second
const DURATION = __ENV.DURATION || "5m"; // durasi test
const PRE_VUS = Number(__ENV.PRE_VUS || "200"); // VU awal yang dialokasikan
const MAX_VUS = Number(__ENV.MAX_VUS || "1000"); // VU maksimum

const USER_ID = __ENV.USER_ID || "42"; // single user yang dites
const CURRENCY = (__ENV.CURRENCY || "USDT").toUpperCase();
const AMOUNT = Number(__ENV.AMOUNT || "1"); // as-is integer (uji coba)

// ---------- OPTIONS ----------
export const options = {
  scenarios: {
    rps: {
      executor: "constant-arrival-rate",
      rate: RATE, // iters per second
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
    },
  },
  thresholds: {
    errors: ["rate==0"],
    grpc_req_duration: ["p(95)<500", "p(99)<2000"],
  },
};

// ---------- METRICS ----------
export const errors = new Rate("errors");

// ---------- PER-VU CLIENT ----------
const client = new grpc.Client();
let connected = false;

export default function () {
  if (!connected) {
    client.connect(TARGET, { plaintext: true, reflect: true }); // pakai server reflection
    connected = true;
  }

  const txId = `u${USER_ID}-${CURRENCY}-vu${__VU}-it${__ITER}-${Date.now()}`;

  let res;
  try {
    res = client.invoke("wallet.v1.WalletService/Deposit", {
      user_id: USER_ID,
      currency: CURRENCY,
      tx_id: txId,
      amount: AMOUNT,
      // network & meta kosong (diabaikan server)
    });
  } catch (e) {
    errors.add(1);
    fail(`grpc invoke error: ${e && e.message ? e.message : e}`);
  }

  const ok = check(res, {
    "transport OK": (r) => r && r.status === grpc.StatusOK,
    // enum bisa muncul sebagai string ("SUCCESS") di xk6-grpc
    "app status SUCCESS": (r) =>
      r &&
      r.message &&
      (r.message.status === "SUCCESS" || r.message.status === 1),
  });

  if (!ok) errors.add(1);
}

export function teardown() {
  try {
    client.close();
  } catch (_) {}
}
