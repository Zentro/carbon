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
	"carbon/internal/api_key"
	"carbon/internal/api_login_key"
	"carbon/internal/client"
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/user"
	"carbon/mysql"
	"carbon/remote"
	"carbon/router"
	"carbon/system"
	"context"
	"errors"
	"fmt"
	"io"
	log2 "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"
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
	slog.Debug("running in debug mode")

	cfg := config.Get()
	ctx := cmd.Context()

	remoteClient := remote.NewClient(
		cfg.Remote.Location,
		cfg.Remote.BridgeKey,
		cfg.Remote.DataKey,
	)

	database, err := mysql.Initialize()
	if err != nil {
		slog.Error("could not initialize database connection", "error", err)
		os.Exit(1)
	}

	rm, err := resource.NewManager(ctx, remoteClient)
	if err != nil {
		slog.Error("could not initialize resource manager", "error", err)
	}
	sm, err := server.NewManager(ctx, database)
	if err != nil {
		slog.Error("could not initialize server manager", "error", err)
	}

	um, err := user.NewManager(ctx, remoteClient)
	if err != nil {
		slog.Error("could not initialize the user manager", "error", err)
	}

	tm, err := api_login_key.NewManager(ctx, database)
	if err != nil {
		slog.Error("could not initialize the token manager", "error", err)
	}

	km, err := api_key.NewManager(ctx, database)
	if err != nil {
		slog.Error("could not initialze the api key manager", "error", err)
	}

	cm, err := client.NewManager(ctx, database)
	if err != nil {
		slog.Error("could not initialze the client manager", "error", err)
	}

	managers := router.ManagerGroup{
		ResourceManager:    rm,
		ServerManager:      sm,
		UserManager:        um,
		ApiLoginKeyManager: tm,
		ApiKeyManager:      km,
		ClientManager:      cm,
	}

	r := router.NewClient(remoteClient, managers)

	slog.Info("starting webserver",
		"use_ssl", config.Get().Api.Ssl.Enabled,
		"use_auto_tls", useAutoTls,
		"host_address", config.Get().Api.Host,
		"host_port", config.Get().Api.Port,
	)

	// Create a new HTTP server instance.
	s := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%d", config.Get().Api.Host, config.Get().Api.Port),
		WriteTimeout: time.Second * config.Get().Api.WriteTimeout,
		ReadTimeout:  time.Second * config.Get().Api.ReadTimeout,
		IdleTimeout:  time.Second * config.Get().Api.IdleTimeout,
	}

	if useAutoTls {
		slog.Info("webserver will start with auto-TLS",
			"hostname", tlsHostname,
		)
		// but it doesn't! not yet at least...
	}

	if config.Get().Api.Ssl.Enabled {
		go func() {
			if err := s.ListenAndServeTLS(config.Get().Api.Ssl.CertificateFile, config.Get().Api.Ssl.KeyFile); err != nil {
				slog.Error("failed to configure TLS webserver", "error", err)
				os.Exit(1)
			}
		}()
		return
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to configure webserver", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("web server started")

	q := make(chan os.Signal, 1)
	// Wait and accept graceful shutdowns when quit via SIGINT (Ctrl+C or DEL)
	// SIGKILL, SIGQUIT, or SIGTERM will not be caught.
	signal.Notify(q, os.Interrupt)

	// Block until a signal is received.
	<-q

	slog.Warn("shutting down...")

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
			log2.Fatal("cmd/root: configuration file not found: ", err)
		}
		log2.Fatal("cmd/root: failed to create the configuration file: ", err)
	}
}

func initLogging() {
	d := config.Get().LogDirectory
	path := filepath.Join(d, "carbon.log")

	// Open the log file and append to it in append mode.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log2.Fatal("cmd/root: failed to open log file: ", err)
	}

	// Log to both stderr and file.
	mw := io.MultiWriter(os.Stderr, file)

	// We will always default to the info log level unless we
	// specify in the config, or it's manually set with the --debug flag
	level := slog.LevelInfo
	if debug || config.Get().Debug {
		level = slog.LevelDebug
	}

	// We no longer use lumberjack for log rotation and instead depend
	// on a logrotate file provided by the user

	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func printLogo() {
	fmt.Printf(colorstring.Color(`[blue]
                   __              
  _________ ______/ /_  ____  ____ 
 / ___/ __ \/ ___/ __ \/ __ \/ __ \
/ /__/ /_/ / /  / /_/ / /_/ / / / /
\___/\__,_/_/  /_.___/\____/_/ /_/   [reset]

Copyright (c) 2022, 2025 Rafael Galvan and contributors.

Rigs of Rods Web API (carbon) [Version %s]

[bold]Use of this source code is governed by the GPLv3 license.
The license can be found in the LICENSE file.
[reset]

Learn more at https://www.rigsofrods.org


`), system.Version)
}
