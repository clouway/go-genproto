package httpkit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/clouway/go-genproto/clouwayapis/rpc/fileserve"
	"github.com/clouway/go-genproto/clouwayapis/rpc/request"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// UnmarshalJSON decodes the message bytes into a protobuf message.
func UnmarshalJSON(b []byte, m proto.Message) error {
	unmarshaller := protojson.UnmarshalOptions{DiscardUnknown: true}
	return unmarshaller.Unmarshal(b, m)
}

// EncodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func EncodeHTTPGenericResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(*fileserve.BinaryFile); ok {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(f.FileName)))
		if f.ContentType == "application/pdf" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", url.QueryEscape(f.FileName)))
		}
		w.Header().Set("Content-Type", f.ContentType)
		w.Write(f.Content)
		return nil
	}

	marshaller := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: false}

	b, err := marshaller.Marshal(response.(proto.Message))
	if err != nil {
		return err
	}
	w.Write(b)
	return nil
}

// HeadersToContext adds all HTTP header values into the passed context.Context. The keys
// are added with request.ContextKey as and lookups should be performed by using the same
// type.
func HeadersToContext(ctx context.Context, r *http.Request) context.Context {
	return HeadersToContextExcluding(ctx, r, []string{})
}

// HeadersToContextExcluding adds all HTTP header values into the passed context.Context. The keys
// are added with request.ContextKey as and lookups should be performed by using the same
// type.
func HeadersToContextExcluding(ctx context.Context, r *http.Request, excludeHeaders []string) context.Context {
	m := make(map[string]bool)
	for _, header := range excludeHeaders {
		m[strings.ToLower(header)] = true
	}

	for k := range r.Header {
		key := strings.ToLower(k)

		_, ok := m[strings.ToLower(k)]
		if ok {
			continue
		}
		// The key is added in strings.ToLower which is the grpc metadata format of the key so
		// that it can be accessed in either format
		ctx = context.WithValue(ctx, request.ContextKey(key), r.Header.Get(k))
	}

	// Tune specific change.
	// also add the request url
	ctx = context.WithValue(ctx, request.ContextKey("request-url"), r.URL.Path)
	ctx = context.WithValue(ctx, request.ContextKey("transport"), "HTTPJSON")

	return ctx
}

// CookiesToContext appends all cookies from the request to Context.
func CookiesToContext(ctx context.Context, r *http.Request) context.Context {
	for _, c := range r.Cookies() {
		key := strings.ToLower(c.Name)
		ctx = context.WithValue(ctx, request.ContextKey(key), c.Value)
	}
	return ctx
}
