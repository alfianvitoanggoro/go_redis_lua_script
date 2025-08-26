import grpc from "k6/net/grpc";
import { check } from "k6";
import { Rate } from "k6/metrics";

// ========= Env params =========
const TARGET = __ENV.TARGET || "localhost:50051";
const RATE = parseInt(__ENV.RATE || "1000", 10); // requests per second
const DURATION = __ENV.DURATION || "5m";
const PRE_VUS = parseInt(__ENV.PRE_VUS || "200", 10);
const MAX_VUS = parseInt(__ENV.MAX_VUS || "1500", 10);
const USERS = parseInt(__ENV.USERS || "10000", 10); // jumlah distinct user_id
const MODE = (__ENV.MODE || "blast").toLowerCase(); // "blast" | "idem"
const MIN_AMOUNT = parseInt(__ENV.MIN_AMOUNT || "1", 10);
const MAX_AMOUNT = parseInt(__ENV.MAX_AMOUNT || "1000000", 10);

// ========= Options =========
export const options = {
  scenarios: {
    rps: {
      executor: "constant-arrival-rate",
      rate: RATE, // req per second
      timeUnit: "1s",
      duration: DURATION,
      preAllocatedVUs: PRE_VUS,
      maxVUs: MAX_VUS,
    },
  },
  thresholds: {
    errors: ["rate==0"],
    // tweak sesuai SLO kamu:
    grpc_req_duration: ["p(95)<500", "p(99)<2000"],
  },
};

// ========= Metrics =========
export const errors = new Rate("errors");

// ========= Data generators =========
const fiat = ["IDR", "USD", "SGD", "EUR"];
const crypto = ["USDT", "BTC", "ETH"];
const allCurrencies = fiat.concat(crypto);

function randInt(min, max) {
  // inclusive
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function pickCurrency() {
  // 50:50 fiat vs crypto
  if (Math.random() < 0.5) {
    return fiat[randInt(0, fiat.length - 1)];
  }
  return crypto[randInt(0, crypto.length - 1)];
}

function randomUserId() {
  // user_id sebagai string (sesuai proto)
  const id = randInt(0, USERS - 1);
  return String(id);
}

// ========= gRPC client (reused per VU) =========
const client = new grpc.Client();
let connected = false;

export default function () {
  if (!connected) {
    // reflect:true agar tidak perlu file .proto saat invoke
    client.connect(TARGET, { plaintext: true, reflect: true });
    connected = true;
  }

  const user_id = randomUserId();
  const currency = pickCurrency();
  const amount = randInt(MIN_AMOUNT, MAX_AMOUNT);

  // tx_id:
  // - mode "idem": tx tetap sama per user+currency → mengetes idemp/FIFO per user
  // - mode "blast": tx unik setiap request
  const tx_id =
    MODE === "idem"
      ? `idem-${user_id}-${currency}`
      : `tx-${user_id}-${currency}-${__VU}-${Date.now()}-${Math.floor(
          Math.random() * 1e9
        )}`;

  let res;
  try {
    res = client.invoke("wallet.v1.WalletService/Deposit", {
      user_id,
      currency,
      tx_id,
      amount, // minor units (contoh: IDR=1 -> 1 rupiah, USD=1 -> 1 cent, BTC=1 -> 1 sat)
      // meta: {}   // optional, bisa diisi kalau servermu butuh
    });
  } catch (e) {
    errors.add(1);
    return; // biarkan k6 lanjut
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
