package main

import (
	"log"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/stopper"
)

func blocklistUpdateScheduler(logger *log.Logger, reloadBlocklist chan Blocklist, stop *stopper.Stopper) {
	logl := logex.Levels(logger)

	defer stop.Done()
	defer logl.Info.Println("Stopped")

	logl.Info.Println("Started")

	daily := time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-stop.Signal:
			return
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

			reloadBlocklist <- *blocklist

			logl.Info.Println("Succeeded")
		}
	}
}
