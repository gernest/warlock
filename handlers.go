package warlock

import (
	"log"
	"net/http"

	"github.com/gernest/render"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/monoculum/formam"
)

// Handlers contains set http facing auth methods
type Handlers struct {
	rendr  *render.Render
	sess   Sess
	ustore UserStore
	cfg    *Config
}

// YoungWarlock initialize and returns a ready to use handler it can be used without any arguments
func YoungWarlock(args ...interface{}) *Handlers {
	var opts render.Options
	var cfg *Config
	var rendr *render.Render

	for _, v := range args {
		switch t := v.(type) {
		case render.Options:
			opts = t
		case *Config:
			cfg = t
		case *render.Render:
			rendr = t

		}
	}
	return warlock(opts, cfg, rendr)
}

func warlock(opts render.Options, cfg *Config, r *render.Render) *Handlers {
	var rendr *render.Render
	c := NewConfig(cfg)
	opt := &sessions.Options{MaxAge: c.SessMaxAge, Path: c.SessPath}
	rendr = render.New(opts)
	if r != nil {
		rendr = r
	}

	return &Handlers{
		rendr:  rendr,
		sess:   NewSessStore(c.DB, "sessions", 100, opt, []byte(c.Secret)),
		ustore: NewUserStore(c.DB, "warlock"),
		cfg:    c,
	}
}

// Register is a http handler for registering new users
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.rendr.HTML(w, http.StatusOK, h.cfg.RegisterTmpl, nil)
		return
	}
	if r.Method == "POST" {
		r.ParseForm()
		user := new(User)
		data := render.NewTemplateData()
		if err := formam.Decode(r.Form, user); err != nil {
			h.rendr.HTML(w, http.StatusInternalServerError, h.cfg.ServerErrTmpl, nil)
			return
		}
		if v := user.Validate(); v != nil {
			data.Add("errors", v)
			h.rendr.HTML(w, http.StatusOK, h.cfg.RegisterTmpl, data)
			return
		}
		if h.ustore.Exist(user) {
			data.Add("error", "user already exist")
			h.rendr.HTML(w, http.StatusOK, h.cfg.RegisterTmpl, data)
			return
		}
		if err := h.ustore.CreateUser(user); err != nil {
			h.rendr.HTML(w, http.StatusInternalServerError, h.cfg.ServerErrTmpl, nil)
			return
		}

		ss, err := h.sess.New(r, h.cfg.SessName)
		if err != nil {
			// TODO (gernest): log this error
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
		// TODO (gernest): log this error
	}
	flash := NewFlash()
	data := render.NewTemplateData()
	if r.Method == "GET" {
		if f := flash.Get(ss); f != nil {
			data.Add("flash", f.Data)
		}
		h.rendr.HTML(w, http.StatusOK, h.cfg.LoginTmpl, data)
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
			h.rendr.HTML(w, http.StatusInternalServerError, h.cfg.ServerErrTmpl, nil)
			return
		}
		if v := lg.Validate(); v != nil {
			data.Add("errors", v)
			h.rendr.HTML(w, http.StatusInternalServerError, h.cfg.LoginTmpl, data)
			return
		}
		user, err := h.ustore.GetUser(lg.Email)
		if err != nil {
			flash.Error("wrong email or password, correct and try again")
			data.Add("flash", flash.Data)
			h.rendr.HTML(w, http.StatusInternalServerError, h.cfg.LoginTmpl, data)
			return
		}
		if err = user.MatchPassword(lg.Password); err != nil {
			flash.Error("wrong email or password, correct and try again")
			data.Add("flash", flash.Data)
			h.rendr.HTML(w, http.StatusOK, h.cfg.LoginTmpl, data)
			return
		}
		ss.Values["user"] = user.Email
		err = ss.Save(r, w)
		if err != nil {
			// TODO (gernest): log this error
		}
		http.Redirect(w, r, h.cfg.LoginRedir, http.StatusFound)
		return
	}

}

// Logout deletes the session
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	ss, err := h.sess.New(r, h.cfg.SessName)
	if err != nil {
		// TODO (gernest): log this error
	}
	err = h.sess.Delete(r, w, ss)
	if err != nil {
		// TODO (gernest): log this error
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

// SessionMiddleware checks for session and addss the user to context
func (h *Handlers) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ss, err := h.sess.New(r, h.cfg.SessName)
		if err != nil {
			// TODO (gernest): log this error
		}
		if !ss.IsNew {
			email := ss.Values["user"].(string)
			usr, err := h.ustore.GetUser(email)
			if err == nil {
				context.Set(r, "user", usr)
			}
			log.Println("sess found")
		}
		next.ServeHTTP(w, r)
	})
}
