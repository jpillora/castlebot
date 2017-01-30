package template

import "encoding/json"

func New() *Template {
	a := &Template{}
	return a
}

type Template struct {
	settings struct {
	}
}

func (a *Template) Get() interface{} {
	return &a.settings
}

func (a *Template) Set(j json.RawMessage) error {
	if err := json.Unmarshal(j, &a.settings); err != nil {
		return err
	}
	//do stuff
	return nil
}
