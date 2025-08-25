import grpc from "k6/net/grpc";
import { check } from "k6";
import { Rate } from "k6/metrics";

/**
 * Env params (bisa diubah saat run):
 * TARGET      : host:port gRPC (default localhost:50051)
 * RATE        : request per detik (default 200)
 * DURATION    : durasi test (default 1m)
 * PRE_VUS     : VU awal (default 50)
 * MAX_VUS     : VU maksimum (default 500)
 * USERS       : banyaknya user unik (default 3)
 * PICK        : "roundrobin" | "random" | "single" (default roundrobin)
 * USER_FIXED  : user id tetap saat PICK=single (default "0")
 * MODE        : "blast" | "idem" (default blast)
 * MIN_AMOUNT  : minimal amount (default 1)
 * MAX_AMOUNT  : maksimal amount (default 1000000)
 */

const TARGET = __ENV.TARGET || "localhost:50051";
const RATE = parseInt(__ENV.RATE || "200", 10);
const DURATION = __ENV.DURATION || "1m";
const PRE_VUS = parseInt(__ENV.PRE_VUS || "50", 10);
const MAX_VUS = parseInt(__ENV.MAX_VUS || "500", 10);
const USERS = parseInt(__ENV.USERS || "3", 10);
const PICK = (__ENV.PICK || "roundrobin").toLowerCase();
const USER_FIXED = __ENV.USER_FIXED || "0";
const MODE = (__ENV.MODE || "blast").toLowerCase();
const MIN_AMOUNT = parseInt(__ENV.MIN_AMOUNT || "1", 10);
const MAX_AMOUNT = parseInt(__ENV.MAX_AMOUNT || "1000000", 10);

export const options = {
  scenarios: {
    rps: {
      executor: "constant-arrival-rate",
      rate: RATE,
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
    },
  },
  thresholds: {
    errors: ["rate==0"],
    // SLO contoh, silakan sesuaikan:
    grpc_req_duration: ["p(95)<500", "p(99)<2000"],
  },
};

export const errors = new Rate("errors");

const fiat = ["IDR", "USD", "SGD", "EUR"];
const crypto = ["USDT", "BTC", "ETH"];
const allCur = fiat.concat(crypto);

function randInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}
function pickCurrency() {
  return Math.random() < 0.5
    ? fiat[randInt(0, fiat.length - 1)]
    : crypto[randInt(0, crypto.length - 1)];
}
function nextUserId() {
  if (PICK === "single") return USER_FIXED;
  if (PICK === "random") return String(randInt(0, USERS - 1));
  // roundrobin per-VU (gunakan __ITER & __VU agar merata)
  const idx = ((__ITER || 0) + (__VU || 0)) % USERS;
  return String(idx);
}

const client = new grpc.Client();
let connected = false;

export default function () {
  if (!connected) {
    client.connect(TARGET, { plaintext: true, reflect: true });
    connected = true;
  }

  const user_id = nextUserId();
  const currency = pickCurrency();
  const amount = randInt(MIN_AMOUNT, MAX_AMOUNT);

  // MODE:
  // - "blast": tx unik, bikin antrian panjang per user → uji FIFO murni
  // - "idem" : tx tetap (per user+currency), uji idemp (nanti kalau tx log diaktifkan)
  const tx_id =
    MODE === "idem"
      ? `idem-${user_id}-${currency}`
      : `u${user_id}-${currency}-vu${__VU}-it${__ITER}-${Date.now()}-${Math.floor(
          Math.random() * 1e6
        )}`;

  let res;
  try {
    res = client.invoke("wallet.v1.WalletService/Deposit", {
      user_id,
      currency,
      tx_id,
      amount, // minor units
    });
  } catch (e) {
    errors.add(1);
    return;
  }

  const ok = check(res, {
    "status OK": (r) => r && r.status === grpc.StatusOK,
  });
  if (!ok) errors.add(1);
}

export function teardown() {
  try {
    client.close();
  } catch (_) {}
}
