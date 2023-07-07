package httpkit_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/clouway/go-genproto/clouwayapis/rpc/errdetails"
	"github.com/clouway/go-genproto/clouwayapis/rpc/fileserve"
	"github.com/clouway/go-genproto/clouwayapis/rpc/httpkit"
	"github.com/clouway/go-genproto/clouwayapis/rpc/request"
)

func TestEncodeHTTPGenericResponse(t *testing.T) {
	protoResponse := &errdetails.ErrorInfo{Reason: "Test Reason"}
	w := httptest.NewRecorder()
	httpkit.EncodeHTTPGenericResponse(context.Background(), w, protoResponse)
	b, _ := ioutil.ReadAll(w.Result().Body)
	body := string(b)
	want := `{"reason":"Test Reason","domain":"","metadata":{}}`

	if body != want {
		t.Errorf("unexpected response of EncodeHTTPGenericResponse:\n- want: %v\n-  got: %v", want, body)
	}
}

func TestEncodeBinaryFile(t *testing.T) {
	protoResponse := &fileserve.BinaryFile{ContentType: "image/jpg", FileName: "MyImage.jpg", Content: []byte("::content::")}

	w := httptest.NewRecorder()
	httpkit.EncodeHTTPGenericResponse(context.Background(), w, protoResponse)

	b, _ := ioutil.ReadAll(w.Result().Body)
	contentDisposition := w.Header().Get("Content-Disposition")
	contentType := w.Header().Get("Content-Type")

	wantContentType := "image/jpg"
	wantContentDisposition := "attachment; filename=MyImage.jpg"
	want := []byte("::content::")

	if !reflect.DeepEqual(want, b) {
		t.Errorf("unexpected binary response of EncodeHTTPGenericResponse:\n- want: %v\n-  got: %v", want, b)
		return
	}

	if contentDisposition != wantContentDisposition {
		t.Errorf("unexpected Content-Disposition header:\n- want: %v\n-  got: %v", wantContentDisposition, contentDisposition)
		return
	}

	if wantContentType != contentType {
		t.Errorf("unexpected Content-Type header:\n- want: %v\n-  got: %v", wantContentType, contentType)
		return
	}
}

func TestEncodePDFDocument(t *testing.T) {
	protoResponse := &fileserve.BinaryFile{ContentType: "application/pdf", FileName: "MyDoc.pdf", Content: []byte("::content::")}

	w := httptest.NewRecorder()
	httpkit.EncodeHTTPGenericResponse(context.Background(), w, protoResponse)

	b, _ := ioutil.ReadAll(w.Result().Body)
	contentDisposition := w.Header().Get("Content-Disposition")
	contentType := w.Header().Get("Content-Type")

	wantContentType := "application/pdf"
	wantContentDisposition := "inline; filename=MyDoc.pdf"
	want := []byte("::content::")

	if !reflect.DeepEqual(want, b) {
		t.Errorf("unexpected binary response of EncodeHTTPGenericResponse:\n- want: %v\n-  got: %v", want, b)
		return
	}

	if contentDisposition != wantContentDisposition {
		t.Errorf("unexpected Content-Disposition header:\n- want: %v\n-  got: %v", wantContentDisposition, contentDisposition)
		return
	}

	if wantContentType != contentType {
		t.Errorf("unexpected Content-Type header:\n- want: %v\n-  got: %v", wantContentType, contentType)
		return
	}
}

func TestEncodeHTTPGenericResponseWithEmptySlice(t *testing.T) {
	protoResponse := &errdetails.BadRequest{Errors: []*errdetails.BadRequest_FieldViolation{}}
	w := httptest.NewRecorder()
	httpkit.EncodeHTTPGenericResponse(context.Background(), w, protoResponse)

	b, _ := ioutil.ReadAll(w.Result().Body)
	body := string(b)
	want := `{"message":"","errors":[]}`

	if body != want {
		t.Errorf("unexpected response of EncodeHTTPGenericResponse:\n- want: %v\n-  got: %v", want, body)
	}
}

func TestHeadersToContext(t *testing.T) {
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Authorization", "Bearer token")
	ctx = httpkit.HeadersToContext(ctx, req)

	got := ctx.Value(request.ContextKey("authorization")).(string)
	want := "Bearer token"
	if want != got {
		t.Errorf("unexpected context value:\n- want: %v\n-  got: %v", want, got)
	}
}

func TestHeadersToContextExcluding(t *testing.T) {
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Authorization", "Bearer token")
	req.Header.Add("Content-Type", "application/json")

	ctx = httpkit.HeadersToContextExcluding(ctx, req, []string{"Content-Type"})

	got := ctx.Value(request.ContextKey("content-type"))

	if got != nil {
		t.Errorf("unexpected value of exluded field:\n- want: %v\n-  got: %v", nil, got)
	}
}

func TestCookiesToContext(t *testing.T) {
	ctx := context.Background()

	req, _ := http.NewRequest("GET", "", nil)
	req.AddCookie(&http.Cookie{Name: "SID", Value: "123"})

	ctx = httpkit.CookiesToContext(ctx, req)
	got := ctx.Value(request.ContextKey("sid"))
	want := "123"
	if got != want {
		t.Errorf("unexpected cookie value in context :\n- want: %v\n-  got: %v", want, got)
	}
}
