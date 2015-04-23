package warlock

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/monoculum/formam"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Hndlers contains set http facing auth methods
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
			log.Println(err)
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
			h.Render(w, h.cfg.RegisterTmpl, data)
			return
		}
		if err := h.ustore.CreateUser(user); err != nil {
			log.Println(err)
			h.Render(w, h.cfg.ServerErrTmpl, nil)
			return
		}
		log.Println(h.cfg.RegRedir)
		http.Redirect(w, r, h.cfg.RegRedir, http.StatusFound)
		return
	}
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	h.Render(w, h.cfg.LoginTmpl, nil)
	return
}

// Render is a helper for rndering templates
func (h *Handlers) Render(w http.ResponseWriter, tmpl string, data interface{}) {
	out := new(bytes.Buffer)
	h.Tmpl.ExecuteTemplate(out, tmpl, data)
	io.Copy(w, out)
}
