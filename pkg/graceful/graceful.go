package graceful

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpillora/overseer"
)

const RestartSignal = syscall.SIGUSR2

// SetupGracefulShutdown sets up signal handler and calls cancel on shutdown
func SetupGracefulShutdown(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh,
		RestartSignal,
		syscall.SIGHUP,
		syscall.SIGTSTP,
		os.Interrupt,
		overseer.SIGTERM,
		overseer.SIGUSR1,
		overseer.SIGUSR2,
		syscall.SIGINT,
	)

	go func() {
		sig := <-sigCh
		log.Printf("🔴 Received signal %v. Initiating shutdown...", sig)
		signal.Stop(sigCh)
		cancel()
	}()
}
