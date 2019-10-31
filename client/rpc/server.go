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
/*
Package server is used to start local socket listener.

It contains three server implementations:
* gRPC server (TODO)
* TCP socket server
* websocket server
*/

package server

import (
	"fmt"

	"github.com/nymtech/nym-validator/client"
	"github.com/nymtech/nym-validator/client/rpc/tcpsocket"
	types "github.com/nymtech/nym-validator/client/rpc/clienttypes"
	"github.com/nymtech/nym-validator/client/rpc/websocket"
	"github.com/nymtech/nym-validator/logger"
)

func NewSocketListener(address, typ string, logger *logger.Logger, c *client.Client) (types.SocketListener, error) {
	var s types.SocketListener
	var err error
	switch typ {
	case "tcp":
		s = tcpsocket.NewSocketServer(address, logger, c)
	case "grpc":
		panic("NOT IMPLEMENTED")
	case "websocket":
		s = websocket.NewSocketServer(address, logger, c)
	default:
		err = fmt.Errorf("unknown server type: %s", typ)
	}
	return s, err
}
