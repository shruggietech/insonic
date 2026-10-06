// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"sync"
	"time"
)

// Keep external work outside catalog transactions while maintaining its claim.
func (s *Service) maintain(ctx context.Context, renew func(context.Context) error) (context.Context, func() error) {
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	joined := make(chan struct{})
	var mu sync.Mutex
	var result error
	go func() {
		defer close(joined)
		ticker := time.NewTicker(s.TTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				if e := renew(work); e != nil {
					mu.Lock()
					result = e
					mu.Unlock()
					cancel()
					return
				}
			}
		}
	}()
	var once sync.Once
	finish := func() error {
		once.Do(func() { close(done); <-joined; cancel() })
		mu.Lock()
		defer mu.Unlock()
		return result
	}
	return work, finish
}
