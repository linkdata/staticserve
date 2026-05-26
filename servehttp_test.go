package staticserve_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linkdata/staticserve"
)

const someText = `The quick brown fox jumps over the lazy dog.`

func Test_ServeHTTP_Raw(t *testing.T) {
	ss, err := staticserve.New("test.txt", []byte(someText))
	if err != nil {
		t.Fatal(err)
	}
	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	ss.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Error(sc)
	}
	b, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte(someText)) {
		t.Error(string(b))
	}
}

func Test_ServeHTTP_GZip(t *testing.T) {
	ss, err := staticserve.New("test.txt", []byte(someText))
	if err != nil {
		t.Fatal(err)
	}
	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rq.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ss.ServeHTTP(rr, rq)
	res := rr.Result()
	if sc := res.StatusCode; sc != http.StatusOK {
		t.Error(sc)
	}
	if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
		t.Error(res.Header)
	}
	b, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, ss.Gz) {
		t.Error("data mismatch")
	}
}

func Test_ServeHTTP_GZipBodyIsComplete(t *testing.T) {
	ss, err := staticserve.New("test.txt", []byte(someText))
	if err != nil {
		t.Fatal(err)
	}
	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rq.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ss.ServeHTTP(rr, rq)
	res := rr.Result()
	if sc := res.StatusCode; sc != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, sc)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err = res.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if got := readGzip(t, b); !bytes.Equal(got, []byte(someText)) {
		t.Fatalf("expected %q, got %q", someText, got)
	}
}

func Test_ServeHTTP_HEAD(t *testing.T) {
	ss, err := staticserve.New("test.txt", []byte(someText))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name           string
		acceptEncoding string
		serve          func(*httptest.ResponseRecorder, *http.Request)
	}{
		{
			name: "direct",
			serve: func(rr *httptest.ResponseRecorder, rq *http.Request) {
				ss.ServeHTTP(rr, rq)
			},
		},
		{
			name:           "mux gzip",
			acceptEncoding: "gzip",
			serve: func(rr *httptest.ResponseRecorder, rq *http.Request) {
				mux := http.NewServeMux()
				mux.Handle("GET /test.txt", ss)
				mux.ServeHTTP(rr, rq)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := httptest.NewRequest(http.MethodHead, "/test.txt", nil)
			if tc.acceptEncoding != "" {
				rq.Header.Set("Accept-Encoding", tc.acceptEncoding)
			}
			rr := httptest.NewRecorder()
			tc.serve(rr, rq)
			res := rr.Result()
			if sc := res.StatusCode; sc != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, sc)
			}
			if cc := res.Header.Get("Cache-Control"); cc != staticserve.HeaderCacheControl[0] {
				t.Fatalf("expected cache-control %q, got %q", staticserve.HeaderCacheControl[0], cc)
			}
			if vary := res.Header.Get("Vary"); vary != staticserve.HeaderVary[0] {
				t.Fatalf("expected vary %q, got %q", staticserve.HeaderVary[0], vary)
			}
			if tc.acceptEncoding == "gzip" {
				if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
					t.Fatalf("expected content-encoding gzip, got %q", ce)
				}
			}
			if rr.Body.Len() != 0 {
				t.Fatalf("expected empty body, got %q", rr.Body.String())
			}
		})
	}
}

func Test_ServeHTTP_AcceptEncodingParsing(t *testing.T) {
	ss, err := staticserve.New("test.txt", []byte(someText))
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name          string
		headers       []string
		wantEncoding  string
		wantBodyPlain bool
	}{
		{name: "gzip", headers: []string{"gzip"}, wantEncoding: "gzip"},
		{name: "case insensitive", headers: []string{"GZip"}, wantEncoding: "gzip"},
		{name: "multiple values", headers: []string{"br", "gzip"}, wantEncoding: "gzip"},
		{name: "q zero", headers: []string{"gzip;q=0"}, wantBodyPlain: true},
		{name: "q zero after br", headers: []string{"br, gzip;q=0"}, wantBodyPlain: true},
		{name: "substring", headers: []string{"xgzip"}, wantBodyPlain: true},
		{name: "identity", headers: []string{"identity"}, wantBodyPlain: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rq := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, hdr := range tc.headers {
				rq.Header.Add("Accept-Encoding", hdr)
			}
			rr := httptest.NewRecorder()
			ss.ServeHTTP(rr, rq)
			res := rr.Result()
			if sc := res.StatusCode; sc != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, sc)
			}
			if got := res.Header.Get("Content-Encoding"); got != tc.wantEncoding {
				t.Fatalf("expected content-encoding %q, got %q", tc.wantEncoding, got)
			}
			b, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if err = res.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if tc.wantBodyPlain {
				if !bytes.Equal(b, []byte(someText)) {
					t.Fatalf("expected plain body %q, got %q", someText, b)
				}
				return
			}
			if !bytes.Equal(b, ss.Gz) {
				t.Fatal("expected gzip body")
			}
		})
	}
}

func Test_ServeHTTP_Errors(t *testing.T) {
	ss := &staticserve.StaticServe{
		Gz: []byte{0},
	}
	rq := httptest.NewRequest(http.MethodPut, "/", nil)
	rr := httptest.NewRecorder()
	ss.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusMethodNotAllowed {
		t.Error(sc)
	}
	if allow := rr.Result().Header.Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("expected Allow %q, got %q", "GET, HEAD", allow)
	}

	rq = httptest.NewRequest(http.MethodGet, "/", nil)
	rr = httptest.NewRecorder()
	ss.ServeHTTP(rr, rq)
	if sc := rr.Result().StatusCode; sc != http.StatusInternalServerError {
		t.Error(sc)
	}
}

func Test_ServeHTTP_JavaScriptContentType_FromGZipInput(t *testing.T) {
	js := []byte("console.log('jaws');")
	ssJS, err := staticserve.New("test.js", js)
	if err != nil {
		t.Fatal(err)
	}
	ss, err := staticserve.New("test.JS.gz", ssJS.Gz)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		AcceptEncoding string
		WantBody       []byte
		WantEncoding   string
	}{
		{WantBody: js},
		{AcceptEncoding: "gzip", WantBody: ss.Gz, WantEncoding: "gzip"},
	}

	for _, tc := range testCases {
		rq := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.AcceptEncoding != "" {
			rq.Header.Set("Accept-Encoding", tc.AcceptEncoding)
		}
		rr := httptest.NewRecorder()
		ss.ServeHTTP(rr, rq)
		res := rr.Result()
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Fatalf("status code %d", sc)
		}
		if got := res.Header.Get("Content-Encoding"); got != tc.WantEncoding {
			t.Fatalf("expected content-encoding %q, got %q", tc.WantEncoding, got)
		}
		if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
			t.Fatalf("expected javascript content type, got %q", ct)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, tc.WantBody) {
			t.Fatal("body mismatch")
		}
		if err = res.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
