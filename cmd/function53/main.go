package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/log/logex"
	"github.com/function61/gokit/os/osutil"
	"github.com/function61/gokit/os/systemdinstaller"
	"github.com/function61/gokit/sync/taskrunner"
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
			rootLogger := logex.StandardLogger()

			osutil.ExitIfError(logic(
				osutil.CancelOnInterruptOrTerminate(rootLogger),
				rootLogger))
		},
	})

	app.AddCommand(writeSystemdFileEntry())
	app.AddCommand(writeDefaultConfigEntry())

	osutil.ExitIfError(app.Execute())
}

func logic(ctx context.Context, rootLogger *log.Logger) error {
	logl := logex.Levels(logex.Prefix("main", rootLogger))

	conf, err := readConfig()
	if err != nil {
		return fmt.Errorf("readConfig: %w", err)
	}

	tasks := taskrunner.New(ctx, rootLogger)

	forwarderPool := NewForwarderPool(
		conf.Endpoints,
		logex.Prefix("forwarderPool", rootLogger))

	dnsHandler := NewDnsQueryHandler(
		forwarderPool,
		*conf,
		makeQueryLogger(conf.LogQueries, logex.Prefix("queryLogger", rootLogger)),
		logex.Prefix("queryHandler", rootLogger))

	tasks.Start("forwarderPool", func(ctx context.Context) error {
		return forwarderPool.Run(ctx)
	})

	tasks.Start("dnsListener", func(ctx context.Context) error {
		return runDnsListener(ctx, dnsHandler, logex.Prefix("dnsListener", rootLogger))
	})

	tasks.Start("metricsServer", func(ctx context.Context) error {
		return metricsServer(ctx, *conf, logex.Prefix("metricsServer", rootLogger))
	})

	if !conf.BlocklistDisableUpdates {
		tasks.Start("blocklistUpdateScheduler", func(ctx context.Context) error {
			return blocklistUpdateScheduler(ctx, func(blockList Blocklist) {
				dnsHandler.replaceBlocklist(blockList)
			}, logex.Prefix("blocklistUpdateScheduler", rootLogger))
		})
	}

	blocklist, err := loadBlocklistAndDownloadIfRequired(logl)
	if err != nil {
		// can't fail-by-stopping here, because blocklist downloading might depend on *our* DNS not being up
		logl.Error.Printf("loadBlocklistAndDownloadIfRequired: %v", err)
	} else {
		dnsHandler.replaceBlocklist(*blocklist)
	}

	logl.Info.Printf("Started %s", dynversion.Version)

	return tasks.Wait()
}

func makeQueryLogger(shouldLog bool, logger *log.Logger) QueryLogger {
	if shouldLog {
		return NewLogQueryLogger(logger)
	} else {
		return NewNilQueryLogger()
	}
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
	install := func() error {
		service := systemdinstaller.SystemdServiceFile(
			"function53",
			tagline,
			systemdinstaller.Args("run"),
			systemdinstaller.Docs("https://github.com/function61/function53"))

		if err := systemdinstaller.Install(service); err != nil {
			return err
		}

		fmt.Println(systemdinstaller.GetHints(service))

		return nil
	}

	return &cobra.Command{
		Use:   "write-systemd-unit-file",
		Short: "Install unit file to start this application on startup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(install())
		},
	}
}
