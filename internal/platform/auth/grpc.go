package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	mdUser      = "x-kmap-user"
	mdRoles     = "x-kmap-roles"
	mdDocAccess = "x-kmap-doc-access"
)

func outgoingContext(ctx context.Context) context.Context {
	principal, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	pairs := []string{mdUser, principal.UserID, mdDocAccess, principal.DocAccess}
	md := metadata.Pairs(pairs...)
	if len(principal.Roles) > 0 {
		md.Set(mdRoles, principal.Roles...)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func principalFromIncoming(ctx context.Context) (Principal, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, false
	}
	users := md.Get(mdUser)
	if len(users) == 0 || users[0] == "" {
		return Principal{}, false
	}
	docAccess := AccessPublic
	if values := md.Get(mdDocAccess); len(values) > 0 && values[0] != "" {
		docAccess = values[0]
	}
	return Principal{UserID: users[0], Roles: md.Get(mdRoles), DocAccess: docAccess}, true
}

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(outgoingContext(ctx), method, req, reply, cc, opts...)
	}
}

func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(outgoingContext(ctx), desc, cc, method, opts...)
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if principal, ok := principalFromIncoming(ctx); ok {
			ctx = WithPrincipal(ctx, principal)
		}
		return handler(ctx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if principal, ok := principalFromIncoming(stream.Context()); ok {
			stream = &principalStream{ServerStream: stream, ctx: WithPrincipal(stream.Context(), principal)}
		}
		return handler(srv, stream)
	}
}

type principalStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (stream *principalStream) Context() context.Context {
	return stream.ctx
}
