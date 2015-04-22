package warlock

import (
	"time"

	valid "github.com/asaskevich/govalidator"
)

type User struct {
	FirstName       string `valid:"alphanum,required"`
	LastName        string `valid:"alphanum,required"`
	Email           string `valid:"email,required"`
	Password        string `valid:"alphanum,required"`
	ConfirmPassword string `valid:"alphanum,required"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

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
		return m
	}
	return nil
}
