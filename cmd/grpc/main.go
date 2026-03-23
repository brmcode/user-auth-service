package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/brmcode/user-auth-service/internal/adapter/grpc"
	"github.com/brmcode/user-auth-service/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c, err := app.Bootstrap(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	userServer := grpc.NewUserServer(c.UserRepo, c.RoleRepo, c.Cache, c.Cfg)
	authServer := grpc.NewAuthServer(c.Cfg, c.UserRepo, c.SessionRepo, c.TokenService)

	listener, err := net.Listen("tcp", ":"+c.Cfg.Grpc.Port)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer, err := grpc.NewServer(c.Cfg, userServer, authServer)
	grpcServer.Serve(listener)
}
