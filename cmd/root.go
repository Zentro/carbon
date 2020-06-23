// Copyright (C) 2022-2023 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"carbon/config"
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/token"
	"carbon/internal/user"
	"carbon/mysql"
	"carbon/remote"
	"carbon/router"
	"carbon/system"
	"context"
	"errors"
	"fmt"
	log2 "log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/multi"
	"github.com/apex/log/handlers/text"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

var rootCmd = &cobra.Command{
	Use: "carbon",
	PreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
		initLogging()
	},
	Run: rootCmdRun,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and quit",
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Printf("v%s\nCopyright (c) 2022 Rafael Galvan\n", system.Version)
	},
}

var (
	debug       = false
	configPath  = config.DefaultLocation
	useAutoTls  = false
	tlsHostname = ""
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "run in debug mode")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultLocation, "set the location for the config file")
	rootCmd.PersistentFlags().BoolVar(&useAutoTls, "auto-tls", false, "generate and manage own SSL certificates using Let's Encrypt")
	rootCmd.PersistentFlags().StringVar(&tlsHostname, "tls-hostname", "", "the FQDN for the generated SSL certificate")

	rootCmd.AddCommand(versionCmd)
}

func rootCmdRun(cmd *cobra.Command, _ []string) {
	printLogo()
	log.Debug("running in debug mode")

	remote := remote.NewClient(config.Get().Remote.Location, config.Get().Remote.Key)

	database, err := mysql.Initialize()
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize database connection")
	}

	rm, err := resource.NewManager(cmd.Context(), remote)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize resource manager")
	}
	sm, err := server.NewManager(cmd.Context(), database)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize server manager")
	}

	um, err := user.NewManager(cmd.Context(), remote)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize the user manager")
	}

	tm, err := token.NewManager(cmd.Context(), database)
	if err != nil {
		log.WithField("error", err).Fatal("could not initialize the token manager")
	}

	managers := router.ManagerGroup{
		ResourceManager: rm,
		ServerManager:   sm,
		UserManager:     um,
		TokenManager:    tm,
	}

	r := router.NewClient(remote, managers)

	asyncCacheRefreshSignal := make(chan struct{})
	asyncTokenPurgeSignal := make(chan struct{})
	go func() {
		for {
			select {
			case <-time.After(1 * time.Minute): // After every 10 minutes
				if err := rm.AsyncRefreshCache(context.Background()); err != nil {
					log.WithField("error", err).Warn("failed to refresh resource cache")
				}

			case <-time.After(1 * time.Hour): // After every 60 minutes
				if err := tm.AsyncPurgeDb(context.Background()); err != nil {
					log.WithField("error", err).Warn("failed to purge token database")
				}

			case <-asyncCacheRefreshSignal:
				log.Info("got cache refresh signal")
				rm.AsyncRefreshCache(context.Background())

			case <-asyncTokenPurgeSignal:
				log.Info("got token purge signal")
				tm.AsyncPurgeDb(context.Background())
			}
		}
	}()

	log.WithFields(log.Fields{
		"use_ssl":      config.Get().Api.Ssl.Enabled,
		"use_auto_tls": useAutoTls,
		"host_address": config.Get().Api.Host,
		"host_port":    config.Get().Api.Port,
	}).Info("starting webserver")

	// Create a new HTTP server instance.
	s := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%d", config.Get().Api.Host, config.Get().Api.Port),
		WriteTimeout: time.Second * config.Get().Api.WriteTimeout,
		ReadTimeout:  time.Second * config.Get().Api.ReadTimeout,
		IdleTimeout:  time.Second * config.Get().Api.IdleTimeout,
	}

	if useAutoTls {
		log.WithField("hostname", tlsHostname).
			Info("webserver will start with auto-TLS")
		// but it doesn't! not yet at least...
	}

	if config.Get().Api.Ssl.Enabled {
		go func() {
			if err := s.ListenAndServeTLS(config.Get().Api.Ssl.CertificateFile, config.Get().Api.Ssl.KeyFile); err != nil {
				log.WithField("error", err).Fatal("failed to configure TLS webserver")
			}
		}()
		return
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithField("error", err).Fatal("failed to configure webserver")
		}
	}()

	log.Info("webserver started")

	q := make(chan os.Signal, 1)
	// Wait and accept graceful shutdowns when quit via SIGINT (Ctrl+C or DEL)
	// SIGKILL, SIGQUIT, or SIGTERM will not be caught.
	signal.Notify(q, os.Interrupt)

	// Block until a signal is received.
	<-q

	log.Warn("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Catch any errors when closing listeners.
	if err := s.Shutdown(ctx); err != nil {
		panic(err)
	}

	// Since we don't have to wait for any other services to finalize, we don't
	// need to block on <-ctx.Done(). It may be needed in the future.
	os.Exit(0)
}

func initConfig() {
	err := config.FromFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			exitWithConfigurationError()
		}
		log2.Fatal("cmd/root: failed to create the configuration file: ", err)
	}
}

func initLogging() {
	d := config.Get().LogDirectory
	// We will always default to the info log level unless we specify in the config,
	// or it's manually set with the --debug flag
	log.SetLevel(log.InfoLevel)
	if debug || config.Get().Debug {
		log.SetLevel(log.DebugLevel)
	}

	p := filepath.Join(d, "carbon.log")
	w := &lumberjack.Logger{
		Filename:   p,
		MaxSize:    10,   // megabytes
		MaxBackups: 5,    // number of backups
		MaxAge:     30,   // days
		Compress:   true, // compress old log files
	}

	log.SetHandler(multi.New(
		text.New(os.Stderr),
		text.New(w),
	))
}

func exitWithConfigurationError() {
	fmt.Println(`Please provide a configuration file using the --config flag.`)
	os.Exit(1)
}

func printLogo() {
	fmt.Printf(`Rigs of Rods Web API [Version %s]`, system.Version)
	fmt.Println()
	fmt.Println(`Copyright 2022-2023 Rafael Galvan. All rights reserved.`)
	fmt.Println()
	fmt.Println(`Use of this source code is governed by the GPLv3 license.`)
	fmt.Println(`The license can be found in the LICENSE file.`)
	fmt.Println()
}
