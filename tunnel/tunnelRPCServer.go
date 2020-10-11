package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"wos2/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type tunnelRPCServer struct {
	pb.UnimplementedTunnelServiceServer
	record         tunnelRecord
	serverHostName string
}

var (
	rpcServer      *grpc.Server
	tunnelIns	tunnelRPCServer
	defaultChannel = 10
)

func StartService(address, port, hostName string, defaultLen int) {
	rpcServer := grpc.NewServer()
	tunnelIns := &tunnelRPCServer{
		record: tunnelRecord{
			id2ins: make(map[string]*tunnelImplement),
		},
		serverHostName: hostName,
	}
	pb.RegisterTunnelServiceServer(rpcServer, tunnelIns)
	lsAddress := fmt.Sprintf("%s:%s", address, port)
	listener, err := net.Listen("tcp4", lsAddress)
	if err != nil {
		log.Fatal("rpc service can not listening the port", err)
	}
	err = rpcServer.Serve(listener)
	if err != nil {
		log.Fatal("rpc server can not start", err)
	}
}

func StopService() {
	if rpcServer != nil {
		rpcServer.GracefulStop()
		for _, v := tunnelIns.record.id2ins{
			v.S
		}
	}
}

func (server *tunnelRPCServer) ListenTunnelMsg(request *pb.TunnelChannelRequest, reponses pb.TunnelService_ListenTunnelMsgServer) error {
	remoteHost := request.GetHost()
	tunnelID := request.GetTunnelName()
	log.Printf("remote:%s request tunnelID:%s listening port", remoteHost, tunnelID)
	server.record.mux.RLock()
	if _, ok := server.record.id2ins[tunnelID]; ok {
		log.Printf("remote:%s request tunnelID:%s search success", remoteHost, tunnelID)
		for k := range server.record.id2ins[tunnelID].listeningPort {
			reponses.Send(&pb.TunnelChannelResponse{
				Host: server.serverHostName,
				Port: k,
			})
		}
		server.record.mux.RUnlock()
		return nil
	}

	server.record.mux.RUnlock()
	//new Tunnel
	server.record.mux.Lock()
	defer server.record.mux.Unlock()
	t := newTunnelImplement(tunnelID, defaultChannel)
	log.Printf("remote:%s request tunnelID:%s create success", remoteHost, tunnelID)
	server.record.id2ins[tunnelID] = t
	for k := range server.record.id2ins[tunnelID].listeningPort {
		reponses.Send(&pb.TunnelChannelResponse{
			Host: server.serverHostName,
			Port: k,
		})
	}
	return nil

}

func (server *tunnelRPCServer) CloseTunnelMsg(ctx context.Context, request *pb.TunnelChannelRequest) (response *pb.TunnelChannelResponse, err error) {
	remoteHost := request.GetHost()
	tunnelID := request.GetHost()
	log.Printf("remote:%s request tunnelID:%s close", remoteHost, tunnelID)
	server.record.mux.RLock()
	if _, ok := server.record.id2ins[tunnelID]; ok {
		server.record.id2ins[tunnelID].Stop()
		server.record.mux.RUnlock()
		server.record.mux.Lock()
		delete(server.record.id2ins, tunnelID)
		server.record.mux.Unlock()
		return &pb.TunnelChannelResponse{
			Host: server.serverHostName,
			Port: "0",
		}, nil
	}
	server.record.mux.RUnlock()

	if ctx.Err() == context.Canceled {
		log.Print("request is canceled")
		return nil, status.Error(codes.Canceled, "request is canceled")
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Print("deadline is exceeded")
		return nil, status.Error(codes.DeadlineExceeded, "deadline is exceeded")
	}

	return &pb.TunnelChannelResponse{
		Host: server.serverHostName,
		Port: "0",
	}, nil
}
