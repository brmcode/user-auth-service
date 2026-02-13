package grpc

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	*grpc.Server
}

func NewServer(config *config.Configuration, userServer *UserServer, authServer *AuthServer) (*GrpcServer, error) {
	server := grpc.NewServer()

	pb.RegisterUserServiceServer(server, userServer)
	pb.RegisterAuthServiceServer(server, authServer)
	reflection.Register(server)

	return &GrpcServer{server}, nil
}

func (g *GrpcServer) Serve(listener net.Listener) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC server listening on %s\n", listener.Addr().String())
		if err := g.Server.Serve(listener); err != nil {
			log.Fatalf("gRPC serve failed: %v", err)
		}
	}()

	<-stop
	log.Println("shutting down gRPC server...")

	// Graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		g.Server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server gracefully stopped")
	case <-time.After(10 * time.Second):
		log.Println("gRPC server shutdown timed out, forcing stop")
		g.Server.Stop()
	}
}
