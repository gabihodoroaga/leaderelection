# leaderlection 

This is a library the implements the leader election algorithm using various different backend implementations: Postgresql, GCS, File

## What is leader election 

Leader election is the process of designating a single process or instance as the organizer of some task 
distributed among several nodes or instances.

The leader election process in the library is quite simple and uses a strongly consistent storage system, like a Postgresql database or Google Cloud Storage. It begins with the creation of a lock object, where the leader updates the current timestamp at regular intervals as a way of informing other replicas regarding its leadership. This lock object which be table row in an sql database, or a file in a cloud blob storage or a file, also holds the identity of the current leader. If the leader fails to update the timestamp within the given interval, it is assumed to have been crashed, which is when the inactive replicas race to acquire leadership by updating the lock with their identity.

Check out this blog for a more detailed information about how this library works [hodo.dev/posts/post-42-leader-election](https://hodo.dev/posts/post-42-leader-election).
## How to use

```go
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
		fmt.Printf("leader outsting %s\n", id)
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
```

## Authors

Gabriel Hodoroaga [hodo.dev](https://hodo.dev)

## Licence

This project is licensed under the terms of the MIT license.
