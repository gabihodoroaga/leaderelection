package leaderelection

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Backend is the interface for backend implementations
type Backend interface {
	// WriteEntry attempts to put/update an entry to the storage
	// using condition to evaluate the TTL.
	// Returns true if the lock was obtained, false if not.
	// If error is not nil mean something unexpected happened, and
	// is not equivalent with false.
	WriteEntry(context.Context, string, time.Duration) (bool, error)
}

// Config holds the configuration for the LeaderManager
type Config struct {
	// Term is the amount of time for a leader to be considered
	// without renewal. After this time the leader is considered dead
	// and the leader election starts again
	Term time.Duration
	// Renew is the time interval at which the leader will try to renew
	// his term
	Renew time.Duration
	// Retry is the amount of time to wait between retries of becoming the leader
	Retry time.Duration
}

// LeaderManager is the implementation of the leader election algorithm
type LeaderManager struct {
	backend  Backend
	config   *Config
	identity string

	// OnOusting is called after the current process is not he leader anymore,
	// either because of an error
	// It provides the identity of the elected leader and a cancel function
	// that can be called to stop the election process.
	// This function blocks the election process so it must always return.
	OnElected func(string, context.CancelFunc)

	// OnElected is called after the current process is elected as the leader.
	// It provides the identity of the elected leader and a cancel function
	// that can be called to stop the election process.
	// This function blocks the election process so it must always return.
	OnOusting func(string, context.CancelFunc)

	// OnError is called when an error occurs during the election process
	// ethier on retry or renew stage
	// It provides the identity of the elected leader the error and a cancel function
	// that can be called to stop the election process.
	// This function blocks the election process so it must always return.
	OnError func(string, error, context.CancelFunc)
}

// NewLeaderManager constructs a LeaderMangager
func NewLeaderManager(backend Backend, config *Config) (*LeaderManager, error) {
	return &LeaderManager{
		backend:  backend,
		config:   config,
		identity: uuid.NewString(),
	}, nil
}

// Start the leader election process and it will
// run in a goroutine until the contex is canceled
func (l *LeaderManager) Start(ctx context.Context) {
	newCtx, cancel := context.WithCancel(ctx)

	go func() {
		for {
			err := l.tryToLock(newCtx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				l.OnError(l.identity, err, cancel)
				time.Sleep(l.config.Retry)
				continue
			}
			// if we got here we are the leader
			if l.OnElected != nil {
				l.OnElected(l.identity, cancel)
			}

			err = l.renewLock(newCtx)
			// if we get here we are not the leader anymore
			if l.OnOusting != nil {
				l.OnOusting(l.identity, cancel)
			}
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				l.OnError(l.identity, err, cancel)
			}
		}
	}()
}

func (l *LeaderManager) tryToLock(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
		success, err := l.backend.WriteEntry(ctx, l.identity, l.config.Term)
		if err != nil {
			return err
		}
		if success {
			return nil
		}
	}

	retryTicker := time.NewTicker(l.config.Retry)
	defer retryTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-retryTicker.C:
			success, err := l.backend.WriteEntry(ctx, l.identity, l.config.Term)
			if err != nil {
				return err
			}
			if success {
				return nil
			}
		}
	}

}

func (l *LeaderManager) renewLock(ctx context.Context) error {
	renewTicker := time.NewTicker(l.config.Renew)
	defer renewTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-renewTicker.C:
			success, err := l.backend.WriteEntry(ctx, l.identity, l.config.Term)
			if err != nil {
				return err
			}
			if !success {
				return nil
			}
		}
	}
}
