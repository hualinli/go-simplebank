package gapi

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hualinli/go-simplebank/pb"
	"github.com/hualinli/go-simplebank/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGatewayMux(ctx context.Context, config utils.Config) (*runtime.ServeMux, error) {
	grpcMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
	)

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, grpcMux, config.RPCServerAddress, dialOpts)
	if err != nil {
		return nil, fmt.Errorf("cannot register gateway handler: %w", err)
	}

	return grpcMux, nil
}

func incomingHeaderMatcher(key string) (string, bool) {
	switch key {
	case "Authorization", "X-Forwarded-For", "X-Real-Ip", "User-Agent":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}
