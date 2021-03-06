package backoff

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

type backoff struct {
	f                          func() (bool, error)
	quit, done                 chan struct{}
	msg                        string
	initialBackoff, maxBackoff time.Duration
}

// Interface does f in a loop, sleeping for initialBackoff between
// each iterations.  If it hits an error, it exponentially backs
// off to maxBackoff.  Backoff will log when it backs off, but
// will stop logging when it reaches maxBackoff.  It will also
// log on first success in the beginning and after errors.
type Interface interface {
	Start()
	Stop()
	SetInitialBackoff(time.Duration)
	SetMaxBackoff(time.Duration)
}

// New makes a new Interface
func New(f func() (bool, error), msg string) Interface {
	return &backoff{
		f:              f,
		quit:           make(chan struct{}),
		done:           make(chan struct{}),
		msg:            msg,
		initialBackoff: 10 * time.Second,
		maxBackoff:     60 * time.Second,
	}
}

func (b *backoff) SetInitialBackoff(d time.Duration) {
	b.initialBackoff = d
}

func (b *backoff) SetMaxBackoff(d time.Duration) {
	b.maxBackoff = d
}

// Stop the backoff, and waits for it to stop.
func (b *backoff) Stop() {
	close(b.quit)
	<-b.done
}

// Start the backoff.  Can only be called once.
func (b *backoff) Start() {
	defer close(b.done)
	backoff := b.initialBackoff
	shouldLog := true

	for {
		done, err := b.f()
		if done {
			return
		}

		if err != nil {
			backoff *= 2
			shouldLog = true
			if backoff > b.maxBackoff {
				backoff = b.maxBackoff
				shouldLog = false
			}
		} else {
			backoff = b.initialBackoff
		}

		if shouldLog {
			if err != nil {
				log.Warnf("Error %s, backing off %s: %s",
					b.msg, backoff, err)
			} else {
				log.Infof("Success %s", b.msg)
			}
		}

		// Re-enable logging if we came from an error (suppressed or not)
		// since we want to log in case a success follows.
		shouldLog = err != nil

		select {
		case <-time.After(backoff):
		case <-b.quit:
			return
		}
	}

}
