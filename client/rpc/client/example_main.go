// THIS IS ENTIRELY FOR TESTING PURPOSES TO DEMONSTRATE THAT YOU CAN COMMUNICATE WITH THE CLIENT ON TCP SOCKET OR THE WEBSOCKET

package main

import (
	"fmt"
	"net"
	"net/url"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	types "github.com/nymtech/nym-validator/client/rpc/clienttypes"
	"github.com/nymtech/nym-validator/client/rpc/utils"
)

func WebSocket() {
	provReq := &types.Request{
		Value: &types.Request_GetProviders{
			GetProviders: &types.RequestGetServiceProviders{},
		},
	}
	provReqBytes, err := proto.Marshal(provReq)
	if err != nil {
		panic(err)
	}

	getCredReq := &types.Request{
		Value: &types.Request_GetCredential{
			GetCredential: &types.RequestGetCredential{
				Value: 1,
			},
		},
	}
	getCredReqBytes, err := proto.Marshal(getCredReq)
	if err != nil {
		panic(err)
	}

	u := url.URL{
		Scheme: "ws",
		Host:   "127.0.0.1:9000",
		Path:   "/coco",
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}

	defer c.Close()

	if err := c.WriteMessage(websocket.BinaryMessage, provReqBytes); err != nil {
		panic(err)
	}

	res := &types.Response{}
	_, resBytes, err := c.ReadMessage()
	if err := proto.Unmarshal(resBytes, res); err != nil {
		panic(err)
	}
	provs := res.Value.(*types.Response_GetProviders).GetProviders.Providers

	if err := c.WriteMessage(websocket.BinaryMessage, getCredReqBytes); err != nil {
		panic(err)
	}

	res = &types.Response{}
	_, resBytes, err = c.ReadMessage()
	if err := proto.Unmarshal(resBytes, res); err != nil {
		panic(err)
	}

	credRes := res.Value.(*types.Response_GetCredential).GetCredential
	fmt.Printf("Credential response: %+v\n", credRes)

	randomizeReq := &types.Request{
		Value: &types.Request_Rerandomize{
			Rerandomize: &types.RequestRerandomize{
				Credential: credRes.Credential,
			},
		},
	}
	randomizeReqBytes, err := proto.Marshal(randomizeReq)
	if err != nil {
		panic(err)
	}

	if err := c.WriteMessage(websocket.BinaryMessage, randomizeReqBytes); err != nil {
		panic(err)
	}

	res = &types.Response{}
	_, resBytes, err = c.ReadMessage()
	if err := proto.Unmarshal(resBytes, res); err != nil {
		panic(err)
	}

	rCred := res.Value.(*types.Response_Rerandomize).Rerandomize.Credential

	spendReq := &types.Request{
		Value: &types.Request_SpendCredential{
			SpendCredential: &types.RequestSpendCredential{
				Credential: rCred,
				Materials:  credRes.Materials,
				Provider:   provs[0],
			},
		},
	}
	spendReqBytes, err := proto.Marshal(spendReq)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Randomized credential: %+v\n", rCred)

	if err := c.WriteMessage(websocket.BinaryMessage, spendReqBytes); err != nil {
		panic(err)
	}

	res = &types.Response{}
	_, resBytes, err = c.ReadMessage()
	if err := proto.Unmarshal(resBytes, res); err != nil {
		panic(err)
	}

	fmt.Printf("spending result: %+v\n", res)
}

func TCPSocket() {
	provReq := &types.Request{
		Value: &types.Request_GetProviders{
			GetProviders: &types.RequestGetServiceProviders{},
		},
	}

	getCredReq := &types.Request{
		Value: &types.Request_GetCredential{
			GetCredential: &types.RequestGetCredential{
				Value: 1,
			},
		},
	}

	flushReq := &types.Request{
		Value: &types.Request_Flush{
			Flush: &types.RequestFlush{},
		},
	}

	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if err := utils.WriteProtoMessage(provReq, conn); err != nil {
		panic(err)
	}

	if err := utils.WriteProtoMessage(flushReq, conn); err != nil {
		panic(err)
	}

	res := &types.Response{}
	if err := utils.ReadProtoMessage(res, conn); err != nil {
		panic(err)
	}

	provs := res.Value.(*types.Response_GetProviders).GetProviders.Providers

	if err := utils.WriteProtoMessage(getCredReq, conn); err != nil {
		panic(err)
	}

	if err := utils.WriteProtoMessage(flushReq, conn); err != nil {
		panic(err)
	}

	res = &types.Response{}
	if err := utils.ReadProtoMessage(res, conn); err != nil {
		panic(err)
	}

	credRes := res.Value.(*types.Response_GetCredential).GetCredential
	fmt.Printf("Credential response: %+v\n", credRes)

	randomizeReq := &types.Request{
		Value: &types.Request_Rerandomize{
			Rerandomize: &types.RequestRerandomize{
				Credential: credRes.Credential,
			},
		},
	}

	if err := utils.WriteProtoMessage(randomizeReq, conn); err != nil {
		panic(err)
	}

	if err := utils.WriteProtoMessage(flushReq, conn); err != nil {
		panic(err)
	}

	res = &types.Response{}
	if err := utils.ReadProtoMessage(res, conn); err != nil {
		panic(err)
	}

	rCred := res.Value.(*types.Response_Rerandomize).Rerandomize.Credential

	spendReq := &types.Request{
		Value: &types.Request_SpendCredential{
			SpendCredential: &types.RequestSpendCredential{
				Credential: rCred,
				Materials:  credRes.Materials,
				Provider:   provs[0],
			},
		},
	}

	fmt.Printf("Randomized credential: %+v\n", rCred)

	if err := utils.WriteProtoMessage(spendReq, conn); err != nil {
		panic(err)
	}

	if err := utils.WriteProtoMessage(flushReq, conn); err != nil {
		panic(err)
	}

	res = &types.Response{}
	if err := utils.ReadProtoMessage(res, conn); err != nil {
		panic(err)
	}

	fmt.Printf("spending result: %+v\n", res)
}

func main() {
	//TCPSocket()
	WebSocket()
}
