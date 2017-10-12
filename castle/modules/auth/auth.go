package auth

import "encoding/json"
import "github.com/jpillora/cookieauth"

func New() *Auth {
	a := &Auth{
		CookieAuth: cookieauth.New(),
	}
	return a
}

type Auth struct {
	CookieAuth *cookieauth.CookieAuth
	settings   struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
}

func (a *Auth) ID() string {
	return "auth"
}

func (a *Auth) Get() interface{} {
	return &a.settings
}

func (a *Auth) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &a.settings); err != nil {
			return err
		}
	}
	a.CookieAuth.SetUserPass(a.settings.User, a.settings.Pass)
	//do stuff
	return nil
}
