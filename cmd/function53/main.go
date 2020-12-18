package main

import (
	"fmt"
	"os"

	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/gokit/systemdinstaller"
	"github.com/spf13/cobra"
)

var tagline = "A DNS server for your LAN that blocks ads/malware and encrypts your DNS traffic"

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   tagline,
		Version: dynversion.Version,
	}

	app.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Runs the program",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			mainInternal()
		},
	})
	app.AddCommand(writeSystemdFileEntry())
	app.AddCommand(writeDefaultConfigEntry())

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mainInternal() {
	rootLogger := logex.StandardLogger()

	logl := logex.Levels(logex.Prefix("main", rootLogger))

	conf, err := readConfig()
	if err != nil {
		logl.Error.Fatalf("readConfig: %v", err)
	}

	blocklist, err := loadBlocklistAndDownloadIfRequired(logl)
	if err != nil {
		logl.Error.Fatalf("loadBlocklistAndDownloadIfRequired: %v", err)
	}

	var queryLogger QueryLogger
	if conf.LogQueries {
		queryLogger = NewLogQueryLogger(logex.Prefix("queryLogger", rootLogger))
	} else {
		queryLogger = NewNilQueryLogger()
	}

	workers := stopper.NewManager()

	forwarderPool := NewForwarderPool(
		conf.Endpoints,
		logex.Prefix("forwarderPool", rootLogger),
		workers.Stopper())

	dnsHandler := NewDnsQueryHandler(
		forwarderPool,
		*conf,
		*blocklist,
		logex.Prefix("queryHandler", rootLogger),
		queryLogger,
		workers.Stopper())

	go func(stop *stopper.Stopper) {
		if err := runServer(dnsHandler, stop); err != nil {
			logl.Error.Fatalf("runServer: %v", err)
		}
	}(workers.Stopper())

	go func(stop *stopper.Stopper) {
		if err := metricsServer(*conf, logex.Prefix("metricsServer", rootLogger), stop); err != nil {
			logl.Error.Fatalf("metricsServer: %v", err)
		}
	}(workers.Stopper())

	if conf.BlocklistEnableUpdates {
		go blocklistUpdateScheduler(
			logex.Prefix("blocklistUpdateScheduler", rootLogger),
			dnsHandler.ReloadBlocklist,
			workers.Stopper())
	}

	logl.Info.Printf("Started %s", dynversion.Version)
	logl.Info.Printf("Got %s; stopping", <-ossignal.InterruptOrTerminate())

	workers.StopAllWorkersAndWait()
}

func loadBlocklistAndDownloadIfRequired(logl *logex.Leveled) (*Blocklist, error) {
	blExists, err := blocklistExists()
	if err != nil {
		return nil, err // error checking for presence of
	}

	if !blExists {
		logl.Info.Println("Downloading blocklist")

		if err := blocklistUpdate(); err != nil {
			return nil, err
		}
	}

	logl.Debug.Println("Loading blocklist from disk")
	list, err := blocklistLoadFromDisk()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func writeSystemdFileEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "write-systemd-unit-file",
		Short: "Install unit file to start this application on startup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			systemdHints, err := systemdinstaller.InstallSystemdServiceFile("function53", []string{"run"}, tagline)
			if err != nil {
				panic(err)
			}

			fmt.Println(systemdHints)
		},
	}
}
