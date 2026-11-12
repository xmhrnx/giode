package support

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xmhrnx/giode/store"
)

func TestSend(t *testing.T) {
	var got Report
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("X-Support-Key")
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"url":"https://github.com/o/r/issues/3","number":3}`))
	}))
	defer srv.Close()

	r := Report{App: "sonora", Title: "  " + strings.Repeat("é", 100) + "  ", Description: "details", Crash: strings.Repeat("x", 40000) + "END"}
	url, err := Send(context.Background(), srv.URL, "k", r)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://github.com/o/r/issues/3" || key != "k" {
		t.Errorf("url %q, key %q", url, key)
	}
	if len(got.Title) > maxTitle || !utf8.ValidString(got.Title) || !strings.HasPrefix(got.Title, "é") {
		t.Errorf("title not cut at a character within %d bytes: %q", maxTitle, got.Title)
	}
	if len(got.Crash) > maxAttachment || !strings.HasSuffix(got.Crash, "END") {
		t.Errorf("crash not cut to its end: %d bytes", len(got.Crash))
	}
}

func TestSendErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"too many reports from this network; try again later"}`))
	}))
	defer srv.Close()
	if _, err := Send(context.Background(), srv.URL, "", Report{App: "a", Title: "t"}); err == nil || !strings.Contains(err.Error(), "too many reports") {
		t.Errorf("the relay's reason isn't passed on: %v", err)
	}
	if _, err := Send(context.Background(), srv.URL, "", Report{App: "a", Title: " "}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a report without a summary: %v", err)
	}
}

func TestSendReasons(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
		want   string
	}{
		{404, "<html><h1>There is nothing here yet</h1></html>", "Couldn't send the report: reason - drift"},
		{401, "", "Couldn't send the report: reason - garnet"},
		{403, "denied", "Couldn't send the report: reason - nimbus"},
		{503, "", "Couldn't send the report: reason - tundra"},
		{418, "", "Couldn't send the report: reason - mosaic"},
		{200, "{}", "Couldn't send the report: reason - comet"},
		{502, `{"error":"harbor","code":"harbor"}`, "Couldn't send the report: reason - harbor"},
		{429, `{"error":"too many reports from this network; try again later"}`, "Couldn't send the report: too many reports from this network; try again later."},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
			w.Write([]byte(c.body))
		}))
		_, err := Send(context.Background(), srv.URL+"/report", "", Report{App: "a", Title: "t"})
		srv.Close()
		if err == nil {
			t.Errorf("%d: no error", c.status)
			continue
		}
		if got := failure(err); got != c.want {
			t.Errorf("%d: %q, want %q", c.status, got, c.want)
		}
		if strings.Contains(err.Error(), "nothing here") || strings.Contains(err.Error(), "denied") {
			t.Errorf("%d: the answer's text is shown: %q", c.status, err)
		}
	}
}

func TestTipLabel(t *testing.T) {
	for _, c := range []struct {
		p    store.Product
		want string
	}{
		{store.Product{Title: "Coffee (Sonora)", Price: "€2.99"}, "Coffee · €2.99"},
		{store.Product{Title: "Big tip", Price: "$9.99"}, "Big tip · $9.99"},
		{store.Product{Price: "$1"}, "$1"},
	} {
		if got := tipLabel(c.p); got != c.want {
			t.Errorf("tipLabel(%+v) = %q, want %q", c.p, got, c.want)
		}
	}
}
