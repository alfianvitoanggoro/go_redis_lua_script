import grpc from "k6/net/grpc";
import { check, fail } from "k6";
import { Rate } from "k6/metrics";

// ========= Params =========
const TARGET = __ENV.TARGET || "localhost:50051";
const VUS = parseInt(__ENV.VUS || "50", 10);
const DURATION = __ENV.DURATION || "1m";
const USERS = parseInt(__ENV.USERS || "100", 10);
const MODE = (__ENV.MODE || "blast").toLowerCase(); // blast | idem
const MIN_AMOUNT = parseInt(__ENV.MIN_AMOUNT || "1", 10);
const MAX_AMOUNT = parseInt(__ENV.MAX_AMOUNT || "1000000", 10);

// k6 options
export let options = { vus: VUS, duration: DURATION };

// metrics
export const errors = new Rate("errors");

const fiatCurrencies = ["IDR", "USD", "SGD"];
const cryptoCurrencies = ["BTC", "ETH", "USDT"];

// generate user ids (9000..9000+USERS-1)
const users = Array.from({ length: USERS }, (_, i) => String(9000 + i));

function randInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// === Per-VU client (init context dieksekusi per VU di k6) ===
const client = new grpc.Client();
let connected = false;

export default function () {
  // connect sekali per VU (reflect:true supaya tak perlu .proto)
  if (!connected) {
    client.connect(TARGET, { plaintext: true, reflect: true });
    connected = true;
  }

  const userId = users[randInt(0, users.length - 1)];
  let currency, network;
  if (Math.random() < 0.5) {
    currency = fiatCurrencies[randInt(0, fiatCurrencies.length - 1)];
    network = "NATIVE";
  } else {
    currency = cryptoCurrencies[randInt(0, cryptoCurrencies.length - 1)];
    network = Math.random() < 0.5 ? "ERC20" : "TRC20";
  }

  const amount = randInt(MIN_AMOUNT, MAX_AMOUNT);
  const txId =
    MODE === "idem"
      ? `idem-${userId}-${currency}`
      : `tx-${userId}-${currency}-${__VU}-${Date.now()}-${Math.random()}`;

  let res;
  try {
    res = client.invoke("wallet.v1.WalletService/Deposit", {
      user_id: userId,
      currency,
      network,
      tx_id: txId,
      amount,
    });
  } catch (e) {
    errors.add(1);
    fail(`grpc dial/invoke error: ${e && e.message ? e.message : e}`);
  }

  const ok = check(res, {
    "status OK": (r) => r && r.status === grpc.StatusOK,
  });
  if (!ok) errors.add(1);
}

export function teardown() {
  client.close();
}
