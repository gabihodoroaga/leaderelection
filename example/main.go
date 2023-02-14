package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	le "github.com/gabihodoroaga/leaderelection"
)

func main() {
	ctx := context.Background()
	b, err := le.NewPostgresqlBackend(ctx, "test-key", le.WithConnString("user=postgres password=password database=leaderelection host=localhost"))
	if err != nil {
		panic(err)
	}

	le, err := le.NewLeaderManager(b, &le.Config{Term: 8 * time.Second, Renew: 4 * time.Second, Retry: 2 * time.Second})
	if err != nil {
		panic(err)
	}

	le.OnElected = func(id string, cancel context.CancelFunc) {
		fmt.Printf("leader elected %s\n", id)
	}
	le.OnOusting = func(id string, cancel context.CancelFunc) {
		fmt.Printf("leader ousting %s\n", id)
	}
	le.OnError = func(id string, err error, cancel context.CancelFunc) {
		fmt.Printf("shit happens for leader %s %v\n", id, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	le.Start(ctx)
	fmt.Printf("leader manager started \n")

	<-ctx.Done()
	stop()
	fmt.Printf("server exit\n")
}
