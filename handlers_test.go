package warlock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

var (
	lPath = "/auth/login"
	rPath = "/auth/register"
	oPath = "/auth/logout"
)

func cleanUp(s string) {
	os.Remove(s)
}
func TestHandlers_Register(t *testing.T) {
	ts, client, _ := testServer(t)
	defer ts.Close()
	defer cleanUp("warlock_test.db")
	reqURL := fmt.Sprintf("%s%s", ts.URL, rPath)

	// Get
	w, err := client.Get(reqURL)
	if err != nil {
		t.Error(err)
	}
	res := new(bytes.Buffer)
	defer w.Body.Close()
	io.Copy(res, w.Body)
	if !strings.Contains(res.String(), "register") {
		t.Errorf("Expected %s to contain register", res.String())
	}

	// POST
	vars := "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
	v, err := url.ParseQuery(vars)
	if err != nil {
		t.Error(err)
	}
	wp, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	res.Reset()
	defer wp.Body.Close()

	io.Copy(res, wp.Body)
	if wp.StatusCode != http.StatusOK {
		t.Errorf("Expected %d actual %d", http.StatusOK, wp.StatusCode)
	}
	if !strings.Contains(res.String(), "login") {
		t.Errorf("Expected %s to contain login", res.String())
	}

	// Already exists
	we, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	res.Reset()
	defer we.Body.Close()

	io.Copy(res, we.Body)
	if we.StatusCode != http.StatusOK {
		t.Errorf("Expected %d actual %d", http.StatusOK, we.StatusCode)
	}
	if !strings.Contains(res.String(), "register") {
		t.Errorf("Expected %s to contain login", res.String())
	}

	// Failure to decode
	vars2 := "firstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
	v, err = url.ParseQuery(vars2)
	if err != nil {
		t.Error(err)
	}
	wf, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	res.Reset()
	defer w.Body.Close()
	io.Copy(res, wf.Body)

	if wf.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500 actual %d", wf.StatusCode)
	}
	if !strings.Contains(res.String(), "crashes") {
		t.Errorf("Expected %s to contain crashes", res.String())
	}

	// Failure to validate
	vars2 = "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=past"
	v, err = url.ParseQuery(vars2)
	if err != nil {
		t.Error(err)
	}
	wv, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	res.Reset()
	defer wv.Body.Close()
	io.Copy(res, wv.Body)

	if wv.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 actual %s", wv.StatusCode)
	}
	if !strings.Contains(res.String(), "should match password") {
		t.Errorf("Expect %s to countain should match password", res.String())
	}

}

func TestHandlers_Login(t *testing.T) {
	ts, client, y := testServer(t)
	defer ts.Close()
	defer cleanUp("warlock_test.db")
	reqURL := fmt.Sprintf("%s%s", ts.URL, lPath)

	// There is no such user yet
	vars := "Email=me@me.com&Password=pass"
	v, err := url.ParseQuery(vars)
	if err != nil {
		t.Error(err)
	}
	w, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	defer w.Body.Close()
	res := new(bytes.Buffer)
	io.Copy(res, w.Body)
	if w.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected %d actual %d", http.StatusInternalServerError, w.StatusCode)
	}
	if !strings.Contains(res.String(), "login") {
		t.Errorf("Expected %s to contain login", res.String())
	}

	// Bad form
	v, err = url.ParseQuery("Emil=me@me.com&Password=pass")
	wi, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	defer wi.Body.Close()
	if wi.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected %d actual %d", http.StatusInternalServerError, wi.StatusCode)
	}

	/// fails to validate
	usr := new(User)
	usr.Email = "me@me.com"
	usr.Password = "pass"
	y.ustore.CreateUser(usr)
	v, err = url.ParseQuery("Email=me@me.com&Password=pass")
	if err != nil {
		t.Error(err)
	}
	wp, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	defer wp.Body.Close()
	if wp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected %d actual %s", http.StatusNotFound, wp.StatusCode)
	}

	// Login
	v, err = url.ParseQuery("Email=me@me.com&Password=pass")
	if err != nil {
		t.Error(err)
	}
	wr, err := client.PostForm(reqURL, v)
	if err != nil {
		t.Error(err)
	}
	if wr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected d actual %d", http.StatusNotFound, wr.StatusCode)
	}

	out, err := client.Get(fmt.Sprintf("%s%s", ts.URL, oPath))
	if err != nil {
		t.Error(err)
	}
	defer out.Body.Close()
	if out.StatusCode != http.StatusNotFound {
		t.Errorf("Expected %d actual %d", http.StatusNotFound, w.StatusCode)
	}

}

func testServer(t *testing.T) (*httptest.Server, *http.Client, *Handlers) {
	cfg := new(Config)
	cfg.DB = "warlock_test.db"
	opts := render.Options{Directory: "fixture"}

	y := YoungWarlock(opts, cfg)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Error(err)
	}
	client := &http.Client{Jar: jar}

	h := mux.NewRouter()
	h.HandleFunc("/auth/register", y.Register).Methods("GET", "POST")
	h.HandleFunc("/auth/login", y.Login).Methods("GET", "POST")
	h.HandleFunc("/auth/logout", y.Logout).Methods("GET", "POST")

	ts := httptest.NewServer(h)
	return ts, client, y
}
