package warlock

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gernest/nutz"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// Sess implements gorilla sessions storage backend interface
type Sess struct {
	store    nutz.Storage
	bucket   string
	options  *sessions.Options
	codecs   []securecookie.Codec
	duration int // Time before the session expires
}

type sessionValue struct {
	Data    string    `json:"data"`
	Expires time.Time `json:"expires"`
}

// NewSessStore creates a new bolt dabase based session store backend
func NewSessStore(db, bucket string, duration int, opts *sessions.Options, secrets ...[]byte) Sess {
	return Sess{
		store:    nutz.NewStorage(db, 0660, nil),
		bucket:   bucket,
		options:  opts,
		codecs:   securecookie.CodecsFromPairs(secrets...),
		duration: duration,
	}
}

// Get fetches a session from the registry
func (s Sess) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New create new session
func (s Sess) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	session.Options = s.options
	session.IsNew = true

	cookie, err := r.Cookie(name)
	if err != nil {
		return session, err
	}
	err = securecookie.DecodeMulti(name, cookie.Value, &session.ID, s.codecs...)
	if err != nil {
		return session, err
	}
	err = s.load(session)
	if err != nil {
		return session, err
	}
	session.IsNew = false
	return session, err
}

// Save persist the session into bolt database
func (s Sess) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	sessID := base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
	if session.ID == "" {
		session.ID = strings.TrimRight(sessID, "=")
	}
	if err := s.save(session); err != nil {
		return err
	}
	e, err := securecookie.EncodeMulti(session.Name(), session.ID, s.codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), e, session.Options))
	return nil
}

// Delete removes session from database and the request
func (s Sess) Delete(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	options := *session.Options
	options.MaxAge = -1
	http.SetCookie(w, sessions.NewCookie(session.Name(), "", &options))
	for k := range session.Values {
		delete(session.Values, k)
	}
	ss := s.store.Delete(s.bucket, session.ID)
	return ss.Error
}

func (s Sess) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.codecs...)
	if err != nil {
		return err
	}
	v, err := json.Marshal(sessionValue{
		Data:    encoded,
		Expires: s.getExpires(session.Options.MaxAge),
	})
	ss := s.store.Create(s.bucket, session.ID, v)
	return ss.Error
}

func (s Sess) load(session *sessions.Session) error {
	v := &sessionValue{}
	ss := s.store.Get(s.bucket, session.ID)
	err := json.Unmarshal(ss.Data, v)
	if err != nil {
		return err
	}
	if v.Expires.Sub(time.Now()) < 0 {
		return errors.New("warlock: session expired")
	}
	err = securecookie.DecodeMulti(session.Name(), v.Data, &session.Values, s.codecs...)
	if err != nil {
		return err
	}
	return nil
}

func (s Sess) getExpires(maxAge int) time.Time {
	if maxAge <= 0 {
		return time.Now().Add(time.Second * time.Duration(s.duration))
	}
	return time.Now().Add(time.Second * time.Duration(maxAge))
}
