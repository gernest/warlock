package warlock

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHandlers(t *testing.T) {
	Convey("Register handler", t, func() {
		files := "fixture/templates/*.html"
		tmpl, err := template.ParseGlob(files)
		cfg := new(Config)
		y := YoungWarlock(tmpl, cfg)
		defer y.sess.store.DeleteDatabase()

		client := &http.Client{}

		h := mux.NewRouter()
		h.HandleFunc("/auth/register", y.Register).Methods("GET", "POST")
		h.HandleFunc("/auth/login", y.Login).Methods("GET", "POST")

		ts := httptest.NewServer(h)
		defer ts.Close()

		regURL := fmt.Sprintf("%s/auth/register", ts.URL)

		So(err, ShouldBeNil)
		So(tmpl, ShouldNotBeNil)
		Convey("register", func() {
			vars := "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
			Convey("Get", func() {
				w, err := client.Get(regURL)
				res := new(bytes.Buffer)
				defer w.Body.Close()
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(res.String(), ShouldContainSubstring, "register")

			})
			Convey("POST", func() {
				v, err := url.ParseQuery(vars)
				w, werr := client.PostForm(regURL, v)

				defer w.Body.Close()

				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(werr, ShouldBeNil)
				So(w.StatusCode, ShouldEqual, 200)
				So(res.String(), ShouldContainSubstring, "login")
				Convey("ALready exists", func() {
					w2, _ := client.PostForm(regURL, v)
					res2 := new(bytes.Buffer)
					io.Copy(res2, w2.Body)
					So(res2.String(), ShouldContainSubstring, "register")
				})
			})

			Convey("Fail to decode", func() {
				vars2 := "firstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
				v, err := url.ParseQuery(vars2)
				w, _ := client.PostForm(regURL, v)
				defer w.Body.Close()
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.StatusCode, ShouldEqual, 500)
				So(res.String(), ShouldContainSubstring, "crashes")
			})
			Convey("Fail to validate", func() {
				vars2 := "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=past"
				v, err := url.ParseQuery(vars2)
				w, _ := client.PostForm(regURL, v)
				defer w.Body.Close()
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.StatusCode, ShouldEqual, 200)
				So(res.String(), ShouldContainSubstring, "should match password")
			})
		})
	})
}
