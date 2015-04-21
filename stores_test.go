package warlock

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/securecookie"

	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSess(t *testing.T) {
	Convey("Sess", t, func() {
		maxAge := 30
		sPath := "/"
		cName := "youngWarlock"
		dbName := "sess_store.db"
		sBucket := "sessions"
		secret := []byte("my-secret")
		opts := &sessions.Options{MaxAge: maxAge, Path: sPath}
		store := NewSessStore(dbName, sBucket, 10, opts, secret)
		defer store.store.DeleteDatabase()

		Convey("New", func() {
			Convey("IsNew", func() {
				req, err := http.NewRequest("GET", "http://www.example.com", nil)
				s, serr := store.New(req, cName)

				So(err, ShouldBeNil)
				So(serr, ShouldEqual, http.ErrNoCookie)
				So(s.IsNew, ShouldBeTrue)
				So(s.Options.Path, ShouldEqual, sPath)

				Convey("Save", func() {
					s.Values["user"] = "youngWarlock"
					w := httptest.NewRecorder()
					err = s.Save(req, w)

					So(err, ShouldBeNil)

					Convey("IsNotNew", func() {
						c, err := securecookie.EncodeMulti(s.Name(), s.ID, securecookie.CodecsFromPairs(secret)...)
						newCookie := sessions.NewCookie(s.Name(), c, opts)
						req.AddCookie(newCookie)
						ns, nerr := store.New(req, cName)

						So(err, ShouldBeNil)
						So(nerr, ShouldBeNil)
						So(ns.IsNew, ShouldBeFalse)

						Convey("Get", func() {
							ss, err := store.Get(req, cName)

							So(err, ShouldBeNil)
							So(ss.IsNew, ShouldBeFalse)
						})
						Convey("Delete", func() {
							err = store.Delete(req, w, s)
							So(err, ShouldBeNil)
						})

					})

				})

			})
		})

	})
}
