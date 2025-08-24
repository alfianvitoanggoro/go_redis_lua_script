package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	walletv1 "grls/pkg/proto/wallet/v1"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "gRPC address host:port")
	user := flag.String("user", "1", "user_id")
	cur := flag.String("cur", "IDR", "currency")
	netw := flag.String("net", "NATIVE", "network")
	mode := flag.String("mode", "blast", "test mode: blast | idem")
	n := flag.Int("n", 200, "number of concurrent requests")
	amt := flag.Int64("amt", 1, "amount (minor units)")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := walletv1.NewWalletServiceClient(conn)

	switch *mode {
	case "idem":
		// kirim N request dgn tx_id yang sama (harus 1 applied, N-1 idempotent)
		txID := fmt.Sprintf("idem-%d", time.Now().UnixNano())
		log.Printf("Running IDEM test: n=%d tx_id=%s", *n, txID)
		var wg sync.WaitGroup
		applied := int64(0)
		for i := 0; i < *n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				res, err := client.Deposit(ctx, &walletv1.DepositRequest{
					UserId:   *user,
					Currency: *cur,
					Network:  *netw,
					TxId:     txID,
					Amount:   *amt,
				})
				if err != nil {
					log.Printf("err: %v", err)
					return
				}
				if res.GetApplied() {
					// tidak pakai atomik karena hanya untuk log (boleh diabaikan)
					applied++
				}
			}()
		}
		wg.Wait()
		log.Printf("IDEMPOTENT test done. applied=%d (expected=1), idempotent=%d", applied, *n-int(applied))

	case "blast":
		// kirim N request dengan tx_id unik (harus semua applied)
		log.Printf("Running BLAST test: n=%d", *n)
		var wg sync.WaitGroup
		applied := int64(0)
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < *n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				txID := fmt.Sprintf("blast-%d-%d", time.Now().UnixNano(), i)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				res, err := client.Deposit(ctx, &walletv1.DepositRequest{
					UserId:   *user,
					Currency: *cur,
					Network:  *netw,
					TxId:     txID,
					Amount:   *amt,
				})
				if err != nil {
					log.Printf("err: %v", err)
					return
				}
				if res.GetApplied() {
					applied++
				}
			}(i)
		}
		wg.Wait()
		log.Printf("BLAST test done. applied=%d (expected=%d)", applied, *n)

	default:
		log.Fatalf("unknown mode: %s", *mode)
	}

	log.Println("Done.")
}
