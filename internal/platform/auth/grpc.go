package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	mdUser      = "x-kmap-user"
	mdRoles     = "x-kmap-roles"
	mdDocAccess = "x-kmap-doc-access"
	mdSignature = "x-kmap-principal-sig"
	mdTimestamp = "x-kmap-principal-ts"
)

const maxPrincipalAge = 5 * time.Minute

func canonicalPrincipal(principal Principal, timestamp string) string {
	return strings.Join([]string{
		principal.UserID,
		principal.DocAccess,
		strings.Join(principal.Roles, ","),
		timestamp,
	}, "\x1f")
}

func signPrincipal(key []byte, principal Principal, timestamp string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(canonicalPrincipal(principal, timestamp)))
	return hex.EncodeToString(mac.Sum(nil))
}

func outgoingContext(ctx context.Context, key []byte) context.Context {
	principal, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	md := metadata.Pairs(mdUser, principal.UserID, mdDocAccess, principal.DocAccess)
	if len(principal.Roles) > 0 {
		md.Set(mdRoles, principal.Roles...)
	}
	if len(key) > 0 {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		md.Set(mdTimestamp, timestamp)
		md.Set(mdSignature, signPrincipal(key, principal, timestamp))
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func principalFromIncoming(ctx context.Context, key []byte) (Principal, bool) {
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
	principal := Principal{UserID: users[0], Roles: md.Get(mdRoles), DocAccess: docAccess}

	if len(key) > 0 && !verifyIncoming(md, principal, key) {
		return Principal{}, false
	}
	return principal, true
}

func verifyIncoming(md metadata.MD, principal Principal, key []byte) bool {
	timestamps := md.Get(mdTimestamp)
	signatures := md.Get(mdSignature)
	if len(timestamps) == 0 || len(signatures) == 0 {
		return false
	}
	seconds, err := strconv.ParseInt(timestamps[0], 10, 64)
	if err != nil {
		return false
	}
	if age := time.Since(time.Unix(seconds, 0)); age < -maxPrincipalAge || age > maxPrincipalAge {
		return false
	}
	expected := signPrincipal(key, principal, timestamps[0])
	return hmac.Equal([]byte(expected), []byte(signatures[0]))
}

func UnaryClientInterceptor(key []byte) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(outgoingContext(ctx, key), method, req, reply, cc, opts...)
	}
}

func StreamClientInterceptor(key []byte) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(outgoingContext(ctx, key), desc, cc, method, opts...)
	}
}

func UnaryServerInterceptor(key []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if principal, ok := principalFromIncoming(ctx, key); ok {
			ctx = WithPrincipal(ctx, principal)
		}
		return handler(ctx, req)
	}
}

func StreamServerInterceptor(key []byte) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if principal, ok := principalFromIncoming(stream.Context(), key); ok {
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
