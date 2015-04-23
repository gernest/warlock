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

func TestUserStore(t *testing.T) {
	Convey("UserStore", t, func() {
		ns := NewUserStore("users.db", "account")
		defer ns.store.DeleteDatabase()

		Convey("Create new user", func() {
			u := new(User)
			v := u.Validate()
			err := ns.CreateUser(u)

			So(v, ShouldNotBeNil)
			So(err, ShouldNotBeNil)

			Convey("A valid user", func() {
				u.FirstName = "young"
				u.LastName = "wrlock"
				u.Email = "example@example.com"
				u.Password = "smash"
				u.ConfirmPassword = "smash"

				v = u.Validate()
				err = ns.CreateUser(u)

				So(v, ShouldBeNil)
				So(err, ShouldBeNil)

				Convey("Already Registered user", func() {
					err := ns.CreateUser(u)

					So(err, ShouldNotBeNil)
				})
				Convey("Password mismatch", func() {
					u.ConfirmPassword = "bogus"

					So(u.Validate(), ShouldNotBeNil)
				})
			})
			Convey("Check validation", func() {
				u.FirstName = "young"
				u.LastName = "wrlock"
				u.Email = "example@example.com"
				u.Password = "smash"

				v = u.Validate()

				So(v, ShouldNotBeNil)
			})

		})
		Convey("Retrieving a user", func() {
			u := &User{
				FirstName:       "young",
				LastName:        "warlock",
				Email:           "wrlock@bigbang,com",
				Password:        "shamsh",
				ConfirmPassword: "smash",
			}

			err := ns.CreateUser(u)

			usr, uerr := ns.GetUser(u.Email)

			So(err, ShouldBeNil)
			So(uerr, ShouldBeNil)
			So(usr.FirstName, ShouldEqual, u.FirstName)

			Convey("Retriving a missing record", func() {
				usr, err = ns.GetUser("bogus@me.com")

				So(usr, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
		Convey("Update User", func() {
			u := &User{
				FirstName:       "young",
				LastName:        "warlock",
				Email:           "updatek@bigbang,com",
				Password:        "shamsh",
				ConfirmPassword: "smash",
			}
			cerr := ns.CreateUser(u)
			usr, err := ns.GetUser(u.Email)

			usr.FirstName = "gernest"
			uerr := ns.UpdateUser(usr)

			gusr, gerr := ns.GetUser(u.Email)

			So(cerr, ShouldBeNil)
			So(err, ShouldBeNil)
			So(gerr, ShouldBeNil)
			So(uerr, ShouldBeNil)
			So(gusr.FirstName, ShouldEqual, usr.FirstName)
			So(ns.Exist(u), ShouldBeTrue)
		})

	})
}
