package httpkit

import (
	"context"
	"encoding/json"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// JSONContentType is the default value of JSON messages recognized
// by the HTTP clients.
const JSONContentType = "application/json; charset=utf-8"

// NewBadRequestError is creating a status error for bad request by formatting
// the passed format and arguments into a messages. The response of this error
// will be:
// 		{"message": "invalid json message '123'"}
func NewBadRequestError(format string, a ...interface{}) error {
	st := status.Newf(codes.InvalidArgument, format, a...)
	return st.Err()
}

// ErrorEncoder writes the error to the ResponseWriter, by default a content
// type of application/json, a body of json with key "message" and the value
// error.Error(), and a status code of 500. If the error implements Headerer,
// the provided headers will be applied to the response. If the error
// implements grpc status.Error then the message and details will be encoded
// as json and will be encoded otherwise json.Marshaler, and the marshaling succeeds, the JSON encoded
// form of the error will be used. If the error implements StatusCoder, the
// provided StatusCode will be used instead of 500.
func ErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", JSONContentType)
	if headerer, ok := err.(httptransport.Headerer); ok {
		for k := range headerer.Headers() {
			w.Header().Set(k, headerer.Headers().Get(k))
		}
	}
	code := http.StatusInternalServerError
	if sc, ok := err.(httptransport.StatusCoder); ok {
		code = sc.StatusCode()
	}
	var body []byte
	if e, ok := status.FromError(err); ok {
		code = httpStatusFromCode(e.Code())
		if len(e.Details()) > 0 {
			marshaller := protojson.MarshalOptions{UseProtoNames: true}
			jsonBody, _ := marshaller.Marshal(e.Details()[0].(proto.Message))
			body = jsonBody
		} else {
			body, _ = json.Marshal(errorWrapper{Message: e.Message()})
		}
	} else if marshaler, ok := err.(json.Marshaler); ok {
		body, _ = marshaler.MarshalJSON()
	} else {
		body, _ = json.Marshal(errorWrapper{Message: err.Error()})
		if marshaler, ok := err.(json.Marshaler); ok {
			if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
				body = jsonBody
			}
		}
	}

	w.WriteHeader(code)
	w.Write(body)
}

type errorWrapper struct {
	Message string `json:"message"`
}

// HttpError satisfies the Headerer and StatusCoder interfaces in
// package github.com/go-kit/kit/transport/http.
// It's used to return user defined Error objects
// in JSON format.
type HttpError struct {
	// The status code that will be used for the response.
	status int

	// ErrHeaders is a map of headers that will be
	// appended to the original headers before all
	// to be send to the client.
	headers map[string][]string

	// A payload to be send as the response body. This field
	// will be serialized as JSON message in MarshalJSON call
	// to be send to the clients.
	payload interface{}
}

// NewHttpError creates an HTTP error with the given status code
// and headers. The headers could be nil or empty if you don't
// want such to be send back to the client.
func NewHttpError(code int, payload interface{}, headers map[string][]string) *HttpError {
	return &HttpError{
		status:  code,
		payload: payload,
		headers: headers,
	}
}

// Error implements the error interface, so the HttpError
// could be used like any error.
func (h HttpError) Error() string {
	return "HttpError"
}

func (h HttpError) StatusCode() int {
	return h.status
}

func (h HttpError) Headers() http.Header {
	return h.headers
}

// MarshalJSON encodes payload property as JSON message
// and returns it. The encoded payload is the message
// that is returned as body from the ErrorEncoder.
func (h HttpError) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.payload)
}

// httpStatusFromCode converts a gRPC error code into the corresponding HTTP response status.
// See: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
func httpStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	// Unknown gRPC error
	return http.StatusInternalServerError
}
