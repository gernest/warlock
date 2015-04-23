package warlock

import (
	"log"
	"time"

	valid "github.com/asaskevich/govalidator"
	"github.com/fatih/structs"
)

// user contain account information of a user
type User struct {
	FirstName       string `valid:"alphanum,required"`
	LastName        string `valid:"alphanum,required"`
	Email           string `valid:"email,required"`
	Password        string `valid:"alphanum,required"`
	ConfirmPassword string `valid:"alphanum,required"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Config a basic configuration settings
type Config struct {
	RegisterTmpl  string
	LoginTmpl     string
	NotFoundTmpl  string
	ServerErrTmpl string
	DB            string
	SessMaxAge    int
	SessPath      string
	RegRedir      string
	LoginRedir    string
	Secret        string
}

// NewConfig initializes configuration
func NewConfig(cfg *Config) *Config {
	if cfg != nil {
		return mergeConfig(cfg, defaultConfig())

	}
	return defaultConfig()
}
func defaultConfig() *Config {
	return &Config{
		RegisterTmpl:  "register.html",
		LoginTmpl:     "login.html",
		NotFoundTmpl:  "404.htl",
		ServerErrTmpl: "500.html",
		DB:            "warlock.db",
		SessMaxAge:    30,
		SessPath:      "/",
		RegRedir:      "/auth/login",
		LoginRedir:    "/",
		Secret:        "My-top-secre",
	}
}

func mergeConfig(src *Config, base *Config) *Config {
	s := src
	d := base
	for _, field := range structs.Fields(s) {
		for _, bfield := range structs.Fields(d) {
			if field.Name() == bfield.Name() && !field.IsZero() {
				err := bfield.Set(field.Value())
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
	return d
}

// Validates checks for the field validation also makes sure password anc ConfirmPassword
// fields match. This should be Called only when creating a new user
func (usr *User) Validate() map[string]string {
	m := make(map[string]string)
	if ok, errs := valid.ValidateStruct(usr); !ok {
		switch e := errs.(type) {
		case valid.Errors:
			for _, v := range e {
				switch ne := v.(type) {
				case valid.Error:
					m[ne.Name] = ne.Error()
				}
			}
		}
		if usr.ConfirmPassword != usr.Password {
			ms := " ,should match password"
			m["ConfirmPassword"] = m["ConfirmPassword"] + ms
		}
		return m
	}
	if usr.ConfirmPassword != usr.Password {
		ms := "ConfirmPassowrd should match password"
		m["ConfirmPassword"] = m["ConfirmPassword"] + ms
		return m
	}
	return nil
}
