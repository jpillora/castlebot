package auth

import "encoding/json"

func New() *Auth {
	a := &Auth{}
	return a
}

type Auth struct {
	settings struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
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
	//do stuff
	return nil
}
