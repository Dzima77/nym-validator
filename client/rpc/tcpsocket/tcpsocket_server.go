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

package tcpsocket

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/nymtech/nym-validator/client"
	types "github.com/nymtech/nym-validator/client/rpc/clienttypes"
	"github.com/nymtech/nym-validator/client/rpc/requesthandler"
	"github.com/nymtech/nym-validator/client/rpc/utils"
	"github.com/nymtech/nym-validator/logger"
	"gopkg.in/op/go-logging.v1"
)

// very heavily inspired by https://github.com/tendermint/tendermint/blob/f7f034a8befeeb84a88ae8f0092f9f465d9a2544/abci/server/socket_server.go
// Apache 2.0 license

type SocketServer struct {
	client   *client.Client
	listener net.Listener
	haltedCh chan struct{}
	haltOnce sync.Once
	log      *logging.Logger

	conns      map[int]net.Conn // in principle there should be only a single one here, unless client used some weird implementation
	connsMutex sync.Mutex
	nextConnID int
	address    string
}

func (s *SocketServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener

	if err := s.client.Start(); err != nil {
		return err
	}

	go s.acceptConnectionsRoutine()
	return nil
}

func (s *SocketServer) Shutdown() {
	s.haltOnce.Do(func() { s.halt() })
}

// calls any required cleanup code
func (s *SocketServer) halt() {
	s.log.Info("Starting graceful shutdown")

	if err := s.listener.Close(); err != nil {
		s.log.Errorf("Error closing listener: %v", err)
	}

	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	for id, conn := range s.conns {
		delete(s.conns, id)
		if err := conn.Close(); err != nil {
			s.log.Errorf("Error closing connection id: %d, conn: %v, err: %v", id, conn, err)
		}
	}

	s.client.Shutdown()

	close(s.haltedCh)
}

func (s *SocketServer) Wait() {
	<-s.haltedCh
}

func (s *SocketServer) addConn(conn net.Conn) int {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()

	connID := s.nextConnID
	s.nextConnID++
	s.conns[connID] = conn

	return connID
}

// deletes conn even if close errs
func (s *SocketServer) rmConn(connID int) error {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()

	conn, ok := s.conns[connID]
	if !ok {
		return fmt.Errorf("connection %d does not exist", connID)
	}

	delete(s.conns, connID)
	return conn.Close()
}

func (s *SocketServer) acceptConnectionsRoutine() {
	for {
		// Accept a connection
		s.log.Info("Waiting for new connection...")
		conn, err := s.listener.Accept()
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				s.log.Errorf("Critical accept failure: %v", err)
				return
			}
			continue
		}

		s.log.Info("Accepted a new connection")
		connID := s.addConn(conn)

		closeConn := make(chan error, 2)             // Push to signal connection closed
		responses := make(chan *types.Response, 100) // A channel to buffer responses

		// Read requests from conn and deal with them
		go s.handleRequests(closeConn, conn, responses)
		// Pull responses from 'responses' and write them to conn.
		go s.handleResponses(closeConn, conn, responses)

		// Wait until signal to close connection
		go s.waitForClose(closeConn, connID)
	}
}

func (s *SocketServer) waitForClose(closeConn chan error, connID int) {
	err := <-closeConn
	switch {
	case err == io.EOF:
		s.log.Errorf("Connection was closed by client")
	case err != nil:
		s.log.Errorf("Connection error: %v", err)
	default:
		// never happens
		s.log.Errorf("Connection was closed.")
	}

	// Close the connection
	if err := s.rmConn(connID); err != nil {
		s.log.Errorf("Error in closing connection: %v", err)
	}
}

// Read requests from conn and deal with them
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *types.Response) {
	bufReader := bufio.NewReader(conn)

	for {
		var req = &types.Request{}
		err := utils.ReadProtoMessage(req, bufReader)
		if err != nil {
			if err == io.EOF {
				closeConn <- err
			} else {
				closeConn <- fmt.Errorf("error reading message: %v", err)
			}
			return
		}
		s.handleRequest(req, responses)
	}
}

func (s *SocketServer) handleRequest(req *types.Request, responses chan<- *types.Response) {
	switch r := req.Value.(type) {
	case *types.Request_GetCredential:
		s.log.Info("Get credential request")
		responses <- requesthandler.HandleGetCredential(r, s.client)
	case *types.Request_SpendCredential:
		s.log.Info("Spend credential request")
		responses <- requesthandler.HandleSpendCredential(r, s.client)
	case *types.Request_GetProviders:
		s.log.Info("Get providers request")
		responses <- requesthandler.HandleGetServiceProviders(r, s.client)
	case *types.Request_Rerandomize:
		s.log.Info("Rerandomize request")
		responses <- requesthandler.HandleRerandomize(r, s.client)
	case *types.Request_Flush:
		responses <- requesthandler.HandleFlush(r)
	default:
		s.log.Info("Unknown request")
		responses <- requesthandler.HandleInvalidRequest()
	}
}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, conn net.Conn, responses <-chan *types.Response) {
	bufWriter := bufio.NewWriter(conn)
	for {
		resp := <-responses
		err := utils.WriteProtoMessage(resp, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("error writing message: %v", err.Error())
			return
		}

		if _, ok := resp.Value.(*types.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("error flushing write buffer: %v", err.Error())
				return
			}
		}
	}
}

func NewSocketServer(address string, logger *logger.Logger, c *client.Client) types.SocketListener {
	s := &SocketServer{
		address:  address,
		listener: nil,
		conns:    make(map[int]net.Conn),
		haltedCh: make(chan struct{}),
		log:      logger.GetLogger("tcp-socket-server"),
		client:   c,
	}

	return s
}
