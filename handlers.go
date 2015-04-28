package warlock

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/monoculum/formam"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Handlers contains set http facing auth methods
type Handlers struct {
	Tmpl   *template.Template
	sess   Sess
	ustore UserStore
	cfg    *Config
}

// YoungWarlock initialize and returns a ready to use handler it can be used without any arguments
func YoungWarlock(args ...interface{}) *Handlers {
	var tmpl *template.Template
	var cfg *Config

	for _, v := range args {
		switch t := v.(type) {
		case *template.Template:
			if tmpl == nil {
				tmpl = t
			}
		case *Config:
			if cfg == nil {
				cfg = t
			}
		}
	}
	return warlock(tmpl, cfg)
}

func warlock(tmpl *template.Template, cfg *Config) *Handlers {
	c := NewConfig(cfg)
	opts := &sessions.Options{MaxAge: c.SessMaxAge, Path: c.SessPath}
	return &Handlers{
		Tmpl:   tmpl,
		sess:   NewSessStore(c.DB, "sessions", 100, opts, []byte(c.Secret)),
		ustore: NewUserStore(c.DB, "warlock"),
		cfg:    c,
	}
}

// Register is a http handler for registering new users
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.Render(w, h.cfg.RegisterTmpl, nil)
		return
	}
	if r.Method == "POST" {
		r.ParseForm()
		user := new(User)
		data := make(map[string]interface{})
		if err := formam.Decode(r.Form, user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Render(w, h.cfg.ServerErrTmpl, nil)
			return
		}
		if v := user.Validate(); v != nil {
			data["errors"] = v
			h.Render(w, h.cfg.RegisterTmpl, data)
			return
		}
		if h.ustore.Exist(user) {
			data[".error"] = "user already exists"
			w.WriteHeader(http.StatusBadRequest)
			h.Render(w, h.cfg.RegisterTmpl, data)
			return
		}
		if err := h.ustore.CreateUser(user); err != nil {
			log.Println(err)
			h.Render(w, h.cfg.ServerErrTmpl, nil)
			return
		}

		ss, err := h.sess.New(r, h.cfg.SessName)
		if err != nil {
			log.Println(err)
		}
		ss.Values["user"] = user.Email
		flash := NewFlash()
		flash.Success("Successfully created your account")
		flash.Add(ss)
		ss.Save(r, w)
		http.Redirect(w, r, h.cfg.RegRedir, http.StatusFound)
		return
	}
}

// Login login users
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	ss, err := h.sess.New(r, h.cfg.SessName)
	if err != nil {
		log.Println(err)
	}
	flash := NewFlash()
	data := make(map[string]interface{})
	if r.Method == "GET" {
		if f := flash.Get(ss); f != nil {
			log.Println(f)
			data["flash"] = f.Data
		}
		h.Render(w, h.cfg.LoginTmpl, data)
		return
	}
	if r.Method == "POST" {
		if !ss.IsNew {
			http.Redirect(w, r, h.cfg.LoginRedir, http.StatusFound)
			return
		}
		r.ParseForm()
		lg := new(LoginForm)
		if err := formam.Decode(r.Form, lg); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			h.Render(w, h.cfg.ServerErrTmpl, nil)
			return
		}
		if v := lg.Validate(); v != nil {
			data["errors"] = v
			h.Render(w, h.cfg.LoginTmpl, data)
			return
		}
		user, err := h.ustore.GetUser(lg.Email)
		flash := NewFlash()
		if err != nil {
			log.Println(err)
			flash.Error("wrong email or password, correct and try again")
			data["flash"] = flash.Data
			h.Render(w, h.cfg.LoginTmpl, data)
			return
		}
		if err = user.MatchPassword(lg.Password); err != nil {
			log.Println(err)
			flash.Error("wrong email or password, correct and try again")
			data["flash"] = flash.Data
			h.Render(w, h.cfg.LoginTmpl, data)
			return
		}
		ss.Values["user"] = user.Email
		err = ss.Save(r, w)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, h.cfg.LoginRedir, http.StatusFound)
		return
	}

}

// Logout deletes the session
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	ss, err := h.sess.New(r, h.cfg.SessName)
	if err != nil {
		log.Println(err)
	}
	err = h.sess.Delete(r, w, ss)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

// Render is a helper for rndering templates
func (h *Handlers) Render(w http.ResponseWriter, tmpl string, data interface{}) {
	out := new(bytes.Buffer)
	h.Tmpl.ExecuteTemplate(out, tmpl, data)
	io.Copy(w, out)
}

// SessionMiddleware checks for session and addss the user to context
func (h *Handlers) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ss, err := h.sess.New(r, h.cfg.SessName)
		if err != nil {
			log.Println(err)
		}
		if !ss.IsNew {
			context.Set(r, "inSession", true)
			email := ss.Values["user"].(string)
			usr, err := h.ustore.GetUser(email)
			if err == nil {
				context.Set(r, "user", usr)
			}
		}
		next.ServeHTTP(w, r)
	})
}
