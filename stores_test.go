package warlock

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var (
	maxAge  = 30
	sPath   = "/"
	cName   = "youngWarlock"
	dbName  = "sess_store.db"
	sBucket = "sessions"
	secret  = []byte("my-secret")
	testURL = "http://www.example.com"
)

func TestSess_New(t *testing.T) {
	store, req := sessSetup(t)
	defer store.store.DeleteDatabase()
	testNewSess(store, req, t)
}

func TestSess_Save(t *testing.T) {
	opts := &sessions.Options{MaxAge: maxAge, Path: sPath}
	store := NewSessStore(dbName, sBucket, 10, opts, secret)
	defer store.store.DeleteDatabase()
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		t.Error(err)
	}
	testSaveSess(store, req, t, "user", "gernest")
}

func TestSess_Get(t *testing.T) {
	opts := &sessions.Options{MaxAge: maxAge, Path: sPath}
	store := NewSessStore(dbName, sBucket, 10, opts, secret)
	defer store.store.DeleteDatabase()
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		t.Error(err)
	}
	s := testSaveSess(store, req, t, "user", "gernest")
	c, err := securecookie.EncodeMulti(s.Name(), s.ID, securecookie.CodecsFromPairs(secret)...)
	if err != nil {
		t.Error(err)
	}
	newCookie := sessions.NewCookie(s.Name(), c, opts)
	req.AddCookie(newCookie)
	s, err = store.New(req, cName)
	if err != nil {
		t.Error(err)
	}
	if s.IsNew {
		t.Errorf("Expected  false, actual %v", s.IsNew)
	}
	ss, err := store.Get(req, cName)
	if err != nil {
		t.Error(err)
	}
	if ss.IsNew {
		t.Errorf("Expected  false, actual %v", ss.IsNew)
	}
	if ss.Values["user"] != "gernest" {
		t.Errorf("Expected gernest, actual %s", ss.Values["user"])
	}
}

func TestSess_Delete(t *testing.T) {
	store, req := sessSetup(t)
	defer store.store.DeleteDatabase()
	s := testSaveSess(store, req, t, "user", "gernest")
	w := httptest.NewRecorder()
	err := store.Delete(req, w, s)
	if err != nil {
		t.Error(err)
	}
}

func TestUserStore(t *testing.T) {
	ns := NewUserStore("users.db", "account")
	defer ns.store.DeleteDatabase()

	usrs := []struct {
		first, last, email string
	}{
		{"geofrey", "ernest", "gernest@home.com"},
		{"young", "warlock", "warlock@home.com"},
	}

	// CreateUser
	for _, usr := range usrs {
		u := new(User)
		u.FirstName = usr.first
		u.LastName = usr.last
		u.Email = usr.email
		err := ns.CreateUser(u)
		if err != nil {
			t.Error(err)
		}
	}

	// GetUser
	for _, usr := range usrs {
		u, err := ns.GetUser(usr.email)
		if err != nil {
			t.Error(err)
		}
		// just for fun
		if !ns.Exist(u) {
			t.Errorf("Expected true actual %v", ns.Exist(u))
		}
		if err == nil {
			if u.FirstName != usr.first {
				t.Errorf("Expected %s actual %s", usr.first, u.FirstName)
			}
			if u.LastName != usr.last {
				t.Errorf("Expected %s actual %s", usr.last, u.LastName)
			}
			if u.Email != usr.email {
				t.Errorf("Expected %s actual %s", usr.email, u.Email)
			}
		}
	}

	// UpdateUser
	for _, usr := range usrs {
		u, err := ns.GetUser(usr.email)
		if err != nil {
			t.Error(err)
		}
		fn := fmt.Sprintf("%s wa", u.FirstName)
		u.FirstName = fn
		err = ns.UpdateUser(u)
		if err != nil {
			t.Error(err)
		}
		if err == nil {
			us, uerr := ns.GetUser(usr.email)
			if uerr != nil {
				t.Error(uerr)
			}
			if uerr == nil {
				if us.FirstName != fn {
					t.Errorf("Expected %s actual %s", fn, us.FirstName)
				}
			}
		}
	}

}
func sessSetup(t *testing.T) (Sess, *http.Request) {
	opts := &sessions.Options{MaxAge: maxAge, Path: sPath}
	store := NewSessStore(dbName, sBucket, 10, opts, secret)
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		t.Error(err)
	}
	return store, req
}
func testNewSess(ss Sess, req *http.Request, t *testing.T) *sessions.Session {
	s, err := ss.New(req, cName)
	if err == nil {
		if !s.IsNew {
			t.Errorf("Expected true actual %v", s.IsNew)
		}
		t.Errorf("Expected \"http: named cookie not present\" actual nil")
	}
	return s
}
func testSaveSess(ss Sess, req *http.Request, t *testing.T, key, val string) *sessions.Session {
	s := testNewSess(ss, req, t)
	s.Values[key] = val
	w := httptest.NewRecorder()
	err := s.Save(req, w)
	if err != nil {
		t.Error(err)
	}
	return s
}
