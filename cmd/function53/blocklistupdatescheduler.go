package main

import (
	"context"
	"log"
	"time"

	"github.com/function61/gokit/log/logex"
)

func blocklistUpdateScheduler(ctx context.Context, replacer func(Blocklist), logger *log.Logger) error {
	logl := logex.Levels(logger)

	daily := time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-daily.C:
			logl.Info.Println("Time to update blocklist")

			if err := blocklistUpdate(); err != nil {
				logl.Error.Printf("Failed: %v", err)
				break
			}

			blocklist, err := blocklistLoadFromDisk()
			if err != nil {
				logl.Error.Printf("Load from disk failed: %v", err)
				break
			}

			replacer(*blocklist)

			logl.Info.Println("Succeeded")
		}
	}
}
