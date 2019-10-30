// run.go - nym socket client startup.
// Copyright (C) 2019  Nym Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/nymtech/nym-validator/client"
	"github.com/nymtech/nym-validator/client/config"
	server "github.com/nymtech/nym-validator/client/rpc"
	"github.com/nymtech/nym-validator/daemon"
	"github.com/nymtech/nym-validator/logger"
	"net"
	"os"
)

const (
	serviceName       = "nym-socket-client"
	defaultConfigFile = "./config.toml"
	localAddress      = "127.0.0.1" // TODO: possibly allow to override this value with a flag?
)

func cmdRunSocket(args []string, usage string) {
	opts := daemon.NewOpts(serviceName, "run [OPTIONS]", usage)

	daemon.Start(func(args []string) daemon.Service {
		cfgFile := opts.Flags("--f").Label("CFGFILE").String("Path to the config file of the client", defaultConfigFile)
		socketType := opts.Flags("--socket").Label("SOCKETTYPE").String("Type of the socket we want to run on (tcp / websocket)")
		port := opts.Flags("--port").Label("PORT").String("Port to listen on")

		params := opts.Parse(args)
		if len(params) != 0 {
			opts.PrintUsage()
			os.Exit(-1)
		}

		cfg, err := config.LoadFile(*cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config file '%v': %v\n", *cfgFile, err)
			os.Exit(-1)
		}

		// Start up the client.
		client, err := client.New(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
			os.Exit(-1)
		}

		// TODO: a better approach to that, but to be honest, we need to rewrite client anyway...
		socketLogger, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)

		socketListener, err := server.NewSocketListener(net.JoinHostPort(localAddress, *port), *socketType, socketLogger, client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to spawn socket listener instance: %v\n", err)
			os.Exit(-1)
		}

		if err := socketListener.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start socket listener instance: %v\n", err)
			os.Exit(-1)
		}

		return socketListener
	}, args)
}
