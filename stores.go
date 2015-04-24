package warlock

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gernest/nutz"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
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

// UserStore user storage stuffs
type UserStore struct {
	store  nutz.Storage
	bucket string
}

type Flash struct {
	Data map[string]interface{}
}

func NewFlash() *Flash {
	return &Flash{Data: make(map[string]interface{})}
}
func (f *Flash) Success(msg string) {
	f.Data["FlashSuccess"] = msg
}

func (f *Flash) Notice(msg string) {
	f.Data["FlashNotice"] = msg
}

func (f *Flash) Error(msg string) {
	f.Data["FlashError"] = msg
}
func (f *Flash) Add(s *sessions.Session) {
	data, err := json.Marshal(f)
	if err == nil {
		s.AddFlash(data)
	}
}

func (f *Flash) Get(s *sessions.Session) *Flash {
	if flashes := s.Flashes(); flashes != nil {
		data := flashes[0]
		if err := json.Unmarshal(data.([]byte), f); err != nil {
			log.Println(err)
			return nil
		}
		return f
	}
	return nil
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

// NewUserStore deals with storage of users
func NewUserStore(db, bucket string) UserStore {
	return UserStore{
		store:  nutz.NewStorage(db, 0600, nil),
		bucket: bucket,
	}
}

// CreateUser creates a new user, email is used as the key.
func (us UserStore) CreateUser(usr *User) error {
	usr.CreatedAt = time.Now()
	p, err := bcrypt.GenerateFromPassword([]byte(usr.Password), 8)
	if err != nil {
		return err
	}
	usr.Password = string(p)
	usr.ConfirmPassword = ""
	data, err := json.Marshal(usr)
	if err != nil {
		return err
	}
	g := us.store.Get(us.bucket, usr.Email)
	if g.Data != nil {
		return errors.New("warlock: email already exists")
	}
	z := g.Create(us.bucket, usr.Email, data)
	return z.Error
}

// GetUser retrives a user given a valid email address
func (us UserStore) GetUser(email string) (*User, error) {
	g := us.store.Get(us.bucket, email)
	if g.Error != nil {
		return nil, g.Error
	}
	usr := new(User)
	err := json.Unmarshal(g.Data, usr)
	if err != nil {
		return nil, err
	}
	return usr, nil
}

// UpdateUser updates user
func (us UserStore) UpdateUser(usr *User) error {
	usr.UpdatedAt = time.Now()
	data, err := json.Marshal(usr)
	if err != nil {
		return err
	}
	up := us.store.Update(us.bucket, usr.Email, data)
	return up.Error
}

// Exists checks if a give user already exists
func (us UserStore) Exist(usr *User) bool {
	g := us.store.Get(us.bucket, usr.Email)
	if g.Data != nil {
		return true
	}
	return false
}
