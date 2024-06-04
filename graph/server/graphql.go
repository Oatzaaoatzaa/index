package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/99designs/gqlgen/graphql/introspection"
	"github.com/memocash/index/graph/generated"
	"github.com/memocash/index/graph/resolver"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"net"
	"net/http"
)

func GetGraphQLHandler() func(w http.ResponseWriter, r *http.Request) {
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{}}))
	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		pathStr := graphql.GetPath(ctx).String()
		if pathStr != "" {
			pathStr = " (" + pathStr + ")"
		}
		resolver.LogContextRequest(ctx, fmt.Sprintf("error processing request%s; %v", pathStr, e))
		var internalError resolver.InternalError
		if errors.As(e, &internalError) {
			return &gqlerror.Error{
				Message: "Internal server error",
				Extensions: map[string]interface{}{
					"code": "INTERNAL_SERVER_ERROR",
				},
			}
		}
		return graphql.DefaultErrorPresenter(ctx, e)
	})
	return func(w http.ResponseWriter, r *http.Request) {
		graphRequest := resolver.NewRequest(getIpAddress(r), "/graphql")
		var finalMessages []string
		if r.Header.Get("Upgrade") != "" {
			finalMessages = append(finalMessages, "[close]")
		}
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Headers", "Content-Type, Server")
		r = r.WithContext(resolver.AttachRequestToContext(r.Context(), graphRequest))
		writer := &sizeWriter{httpWriter: w}
		srv.ServeHTTP(writer, r)
		graphRequest.Size = writer.totalSize
		graphRequest.LogFinal(finalMessages...)
	}
}

func getIpAddress(r *http.Request) string {
	cfIp := r.Header.Get("CF-Connecting-IP")
	if cfIp != "" {
		return cfIp
	}
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	remoteHost, _, _ := net.SplitHostPort(r.RemoteAddr)
	return remoteHost
}
