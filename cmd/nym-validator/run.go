// run.go - validator startup definition.
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
	"os"

	"github.com/nymtech/nym-validator/daemon"
	"github.com/nymtech/nym-validator/server/config"
	"github.com/nymtech/nym-validator/server/issuer"
)

const (
	serviceName       = "nym-validator-demo"
	defaultConfigFile = "/issuer/config.toml"
)

func cmdRun(args []string, usage string) {
	opts := daemon.NewOpts(serviceName, "run [OPTIONS]", usage)

	daemon.Start(func(args []string) daemon.Service {
		cfgFile := opts.Flags("--f").Label("CFGFILE").String("Path to the config file of the validator", defaultConfigFile)

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

		// Start up the issuer.
		issuer, err := issuer.New(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to spawn issuer instance: %v\n", err)
			os.Exit(-1)
		}

		return issuer
	}, args)
}
