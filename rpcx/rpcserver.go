package rpcx

import (
	"net"

	"time"

	"github.com/weblazy/core/logx"
	"github.com/weblazy/core/rpcx/auth"
	"github.com/weblazy/core/rpcx/interceptors"
	"github.com/weblazy/core/rpcx/serverinterceptors"
	"github.com/weblazy/core/system"
	"google.golang.org/grpc"
)

type (
	RpcServer struct {
		*baseRpcServer
		register RegisterFn
	}
)

func init() {
	InitLogger()
}

func NewRpcServer(c RpcServerConf, register RegisterFn) (*RpcServer, error) {
	var err error
	if err = c.Validate(); err != nil {
		return nil, err
	}
	server := &RpcServer{
		baseRpcServer: newBaseRpcServer(c.ListenOn),
		register:      register,
	}
	if err = setupInterceptors(server, c); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *RpcServer) Start() {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		logx.Fatal(err)
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		serverinterceptors.UnaryCrashInterceptor(),
		serverinterceptors.UnaryStatInterceptor(),
	}
	unaryInterceptors = append(unaryInterceptors, s.unaryInterceptors...)
	streamInterceptors := []grpc.StreamServerInterceptor{
		serverinterceptors.StreamCrashInterceptor,
	}
	streamInterceptors = append(streamInterceptors, s.streamInterceptors...)
	options := append(s.options, WithUnaryServerInterceptors(unaryInterceptors...),
		WithStreamServerInterceptors(streamInterceptors...))
	server := grpc.NewServer(options...)
	s.register(server)
	// we need to make sure all others are wrapped up
	// so we do graceful stop at shutdown phase instead of wrap up phase
	shutdownCalled := system.AddShutdownListener(func() {
		server.GracefulStop()
	})
	err = server.Serve(lis)
	shutdownCalled()

	logx.Fatal(err)
}

func setupInterceptors(server *RpcServer, c RpcServerConf) error {
	if c.Timeout > 0 {
		server.AddUnaryInterceptors(serverinterceptors.UnaryTimeoutInterceptor(
			time.Duration(c.Timeout) * time.Millisecond))
	}

	if c.Auth {
		authenticator, err := auth.NewAuthenticator(c.Redis.NewRedis(), c.Redis.Key, c.StrictControl)
		if err != nil {
			return err
		}

		server.AddStreamInterceptors(interceptors.StreamAuthorizeInterceptor(authenticator))
		server.AddUnaryInterceptors(interceptors.UnaryAuthorizeInterceptor(authenticator))
	}

	return nil
}

func (rs *RpcServer) Stop() {
	logx.Close()
}
