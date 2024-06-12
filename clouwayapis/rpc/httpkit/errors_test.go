package httpkit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clouway/go-genproto/clouwayapis/rpc/errdetails"
	"github.com/clouway/go-genproto/clouwayapis/rpc/httpkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type input struct {
	code    codes.Code
	message string
	details *errdetails.BadRequest
}

type want struct {
	headers map[string]string
	status  int
	body    string
}

type test struct {
	name  string
	input input
	want  want
}

type appError struct {
	ErrorCode     int            `json:"errorCode"`
	ErrorMessages []errorMessage `json:"errorMessages"`
}

type errorMessage struct {
	LanguageCode string `json:"languageCode"`
	ErrorMessage string `json:"errorMessage"`
}

func TestEncodeJsonError(t *testing.T) {
	rec := httptest.NewRecorder()
	httpErr := httpkit.NewHttpError(
		http.StatusBadRequest,
		appError{
			ErrorCode:     http.StatusBadRequest,
			ErrorMessages: []errorMessage{{"en", "english error message"}},
		},
		nil,
	)

	httpkit.ErrorEncoder(context.Background(), httpErr, rec)

	body, _ := io.ReadAll(rec.Body)

	gotBody := string(body)
	wantBody := `{"errorCode":400,"errorMessages":[{"languageCode":"en","errorMessage":"english error message"}]}`
	if wantBody != gotBody {
		t.Errorf("unexpected body:\n- want: %v\n-  got: %v", wantBody, gotBody)
	}
}

func TestEncodeProtoError(t *testing.T) {
	tests := []test{
		{
			name:  "not found",
			input: input{code: codes.NotFound, message: "not found", details: nil},
			want: want{
				headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
				status:  http.StatusNotFound,
				body:    `{"message":"not found"}`,
			},
		},
		{
			name:  "already exists",
			input: input{code: codes.AlreadyExists, message: "already exists", details: nil},
			want: want{
				headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
				status:  http.StatusConflict,
				body:    `{"message":"already exists"}`,
			},
		},
		{
			name:  "already exists with details",
			input: input{code: codes.AlreadyExists, details: &errdetails.BadRequest{Message: "item already added"}},
			want: want{
				headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
				status:  http.StatusConflict,
				body:    `{"message":"item already added"}`,
			},
		},
		{
			name: "already exists with details and field violation",
			input: input{
				code: codes.AlreadyExists,
				details: &errdetails.BadRequest{
					Message: "item already added",
					Errors: []*errdetails.BadRequest_FieldViolation{
						{Reason: "Item with id '123' already exists", Field: "itemId"},
					},
				},
			},
			want: want{
				headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
				status:  http.StatusConflict,
				body:    `{"message":"item already added", "errors":[{"reason":"Item with id '123' already exists", "field":"itemId"}]}`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := status.New(test.input.code, test.input.message)
			if test.input.details != nil {
				st, _ = st.WithDetails(test.input.details)
			}

			rec := httptest.NewRecorder()
			httpkit.ErrorEncoder(context.Background(), st.Err(), rec)

			for whn, whv := range test.want.headers {
				headerValue := rec.Header().Get(whn)
				if headerValue != whv {
					t.Errorf("unexpected header returned:\n- want: %v\n-  got: %v", whv, headerValue)
					return
				}
			}
			if rec.Result().StatusCode != test.want.status {
				t.Errorf("unexpected status code:\n- want: %v\n-  got: %v", test.want.status, rec.Result().StatusCode)
				return
			}
			body, _ := io.ReadAll(rec.Body)
			sbody := string(body)
			if sbody != test.want.body {
				t.Errorf("unexpected body:\n- want: %v\n-  got: %v", test.want.body, sbody)
			}
		})
	}
}
