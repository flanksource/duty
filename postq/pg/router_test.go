package pg

import (
	"sync"
	"testing"
	"time"
)

func TestPGRouter(t *testing.T) {
	// Create & run the router
	r := NewNotifyRouter()
	pgNotifyChan := make(chan string)
	go func() {
		r.consume(pgNotifyChan)
	}()

	// Two subscribers
	alpha := r.GetOrCreateChannel("alphaA", "alphaB")
	beta := r.GetOrCreateChannel("beta")

	var alphaCount, betaCount int
	timeout := time.NewTimer(time.Second * 3)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-alpha:
				alphaCount++
				if alphaCount+betaCount == 3 {
					return
				}

			case <-beta:
				betaCount++
				if alphaCount+betaCount == 3 {
					return
				}

			case <-timeout.C:
				return
			}
		}
	}()

	// Simulate receiving pg notify
	go func() {
		pgNotifyChan <- "alphaA 1"
		pgNotifyChan <- "beta 1"
		pgNotifyChan <- "alphaB 1"
	}()

	wg.Wait()
	if alphaCount != 2 {
		t.Errorf("Expected alphaCount to be 2, got %d", alphaCount)
	}

	if betaCount != 1 {
		t.Errorf("Expected betaCount to be 1, got %d", betaCount)
	}
}
