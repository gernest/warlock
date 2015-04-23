package warlock

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHandlers(t *testing.T) {
	Convey("Handlers", t, func() {
		files := "fixture/templates/*.html"

		tmpl, err := template.ParseGlob(files)

		So(err, ShouldBeNil)
		So(tmpl, ShouldNotBeNil)
		Convey("register", func() {
			h := YoungWarlock(tmpl)
			defer h.sess.store.DeleteDatabase()
			vars := "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
			url := "http://www.example.com"

			Convey("Get", func() {
				req, err := http.NewRequest("GET", url, nil)
				w := httptest.NewRecorder()
				h.Register(w, req)
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 200)
				So(res.String(), ShouldContainSubstring, "register")

			})
			Convey("POST", func() {
				req, err := http.NewRequest("POST", fmt.Sprintf("%s?%s", url, vars), nil)
				w := httptest.NewRecorder()
				h.Register(w, req)
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 200)
				So(res.String(), ShouldContainSubstring, "login")
				Convey("ALready exists", func() {
					req2, _ := http.NewRequest("POST", fmt.Sprintf("%s?%s", url, vars), nil)
					w2 := httptest.NewRecorder()
					res2 := new(bytes.Buffer)
					h.Register(w2, req2)
					io.Copy(res2, w2.Body)

					So(res2.String(), ShouldContainSubstring, "register")

				})
			})
			Convey("Fail to decode", func() {
				vars2 := "firstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=pass"
				req, err := http.NewRequest("POST", fmt.Sprintf("%s?%s", url, vars2), nil)
				w := httptest.NewRecorder()
				h.Register(w, req)
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 500)
				So(res.String(), ShouldContainSubstring, "crashes")
			})
			Convey("Fail to validate", func() {
				vars2 := "FirstName=young&LastName=warlock&Email=me@me.com&Password=pass&ConfirmPassword=past"
				req, err := http.NewRequest("POST", fmt.Sprintf("%s?%s", url, vars2), nil)
				w := httptest.NewRecorder()
				h.Register(w, req)
				res := new(bytes.Buffer)
				io.Copy(res, w.Body)

				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 200)
				So(res.String(), ShouldContainSubstring, "should match password")
			})
		})
	})
}
