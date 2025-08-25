import grpc from "k6/net/grpc";
import { check } from "k6";

// Jalankan 10 VU, masing2 1 request -> total 10 request, paralel.
export const options = {
  scenarios: {
    tenParallel: {
      executor: "per-vu-iterations",
      vus: 10,
      iterations: 1,
      maxDuration: "1m",
    },
  },
};

// Env (bisa override saat run)
const TARGET = __ENV.TARGET || "localhost:50051";
const USER_ID = __ENV.USER_ID || "42";
const CURRENCY = __ENV.CURRENCY || "USDT";

export default function () {
  const client = new grpc.Client();
  client.connect(TARGET, { plaintext: true, reflect: true });

  // amount = 1..10 sesuai nomor VU
  const amount = __VU;
  const tx_id = `fifo-demo-${USER_ID}-${CURRENCY}-amt${amount}-${Date.now()}-${Math.floor(
    Math.random() * 1e6
  )}`;

  const res = client.invoke("wallet.v1.WalletService/Deposit", {
    user_id: USER_ID,
    currency: CURRENCY,
    tx_id,
    amount, // integer as-is
  });

  check(res, { "grpc OK": (r) => r && r.status === grpc.StatusOK });

  // Log ringkas biar kelihatan request mana yang sukses
  console.log(
    `sent amount=${amount} tx_id=${tx_id} => grpc=${
      res.status
    } body=${JSON.stringify(res.message)}`
  );

  client.close();
}
