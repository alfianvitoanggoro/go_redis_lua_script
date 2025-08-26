import grpc from "k6/net/grpc";
import { check } from "k6";
import { Rate, Trend } from "k6/metrics";

// ==== ENV ====
const TARGET = __ENV.TARGET || "localhost:50051";
const USER_ID = __ENV.USER_ID || "42";
const CURRENCY = __ENV.CURRENCY || "USDT";

const RATE = parseInt(__ENV.RATE || "1000", 10); // req/s
const TOTAL = parseInt(__ENV.TOTAL || "100000", 10); // total request
const PRE_VUS = parseInt(__ENV.PRE_VUS || "200", 10);
const MAX_VUS = parseInt(__ENV.MAX_VUS || "2000", 10);

// Durasi otomatis = TOTAL / RATE (dibulatkan ke atas)
const seconds = Math.ceil(TOTAL / RATE);

// ==== OPTIONS ====
export const options = {
  scenarios: {
    rps: {
      executor: "constant-arrival-rate",
      rate: RATE,
      timeUnit: "1s",
      duration: `${seconds}s`,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
      gracefulStop: "0s",
    },
  },
  thresholds: {
    errors: ["rate==0"],
  },
};

// ==== METRICS ====
export const errors = new Rate("errors");
export const fifo_wait = new Trend("fifo_wait_ms");

// ==== GRPC CLIENT (per VU) ====
const client = new grpc.Client();
let connected = false;

export default function () {
  if (!connected) {
    client.connect(TARGET, { plaintext: true, reflect: true });
    connected = true;
  }

  const amount = 1; // as-is
  const tx_id = `u${USER_ID}-${CURRENCY}-vu${__VU}-it${__ITER}-${Date.now()}-${Math.floor(
    Math.random() * 1e6
  )}`;

  const t0 = Date.now();
  let res;
  try {
    res = client.invoke("wallet.v1.WalletService/Deposit", {
      user_id: USER_ID,
      currency: CURRENCY,
      tx_id,
      amount,
    });
  } catch (e) {
    errors.add(1);
    return;
  } finally {
    fifo_wait.add(Date.now() - t0);
  }

  const ok = check(res, {
    "grpc OK": (r) => r && r.status === grpc.StatusOK,
    // enum bisa tampil sebagai string "SUCCESS" atau angka 1 tergantung versi/xk6-grpc
    "app success": (r) =>
      r &&
      r.message &&
      (r.message.status === "SUCCESS" ||
        r.message.status === 1 ||
        r.message.Status === "SUCCESS" ||
        r.message.Status === 1),
  });
  if (!ok) errors.add(1);
}

export function teardown() {
  try {
    client.close();
  } catch (_) {}
}
