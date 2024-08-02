package middleware

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Logging(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	const op = "interceptor.Logging"
	//md, ok := metadata.FromIncomingContext(ctx)
	//if ok {
	//	header := md.Get("x-my-header")
	//	log.Printf("[%s] header: %v\n", op, header)
	//}

	message, ok := req.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to convert request to proto.Message")
	}

	raw, err := protojson.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proto message: %w", err)
	}

	log.Printf("[%s] start: %v, %v\n", op, info.FullMethod, string(raw))

	resp, err = handler(ctx, req)
	if err != nil {
		log.Printf("[%s] error:%v\n", op, err.Error())
		return nil, err
	}

	log.Printf("[%s] end\n", op)

	return
}
