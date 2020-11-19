package main

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

const hcsStreamingSubscriptionStatusTickerInterval = 5 * time.Second
const hcsSubscriptionStatusSleepInterval = 250 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context
)

func init() {
	if baselineMessageAdminKey != nil {
		err := createBaselineMessageTopic(*baselineMessageAdminKey, 10)
		if err != nil {
			log.Panicf("failed to initialize dedicated HCS subscription consumer; %s", err.Error())
		}
	}

	err := subscribe(nil) // TODO: use last anchor
	if err != nil {
		log.Panicf("failed to initialize dedicated HCS subscription consumer; %s", err.Error())
	}
}

func main() {
	log.Debug("installing signal handlers for dedicated HCS subscription consumer")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

	log.Debugf("running dedicated HCS subscription consumer main()")
	timer := time.NewTicker(hcsStreamingSubscriptionStatusTickerInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// TODO: check HCS subscription liveness?
		case sig := <-sigs:
			log.Infof("received signal: %s", sig)
			log.Warningf("HCS connection subscriptions are not yet being drained...")
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			time.Sleep(hcsSubscriptionStatusSleepInterval)
		}
	}

	log.Debug("exiting dedicated HCS subscription consumer main()")
	cancelF()
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		log.Debug("shutting down dedicated HCS subscription consumer")
		cancelF()
	}
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}