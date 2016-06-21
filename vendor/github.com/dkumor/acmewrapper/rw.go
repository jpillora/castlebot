package acmewrapper

import (
	"errors"
	"io/ioutil"
)

var ErrNotHandled = errors.New("not handled")

func (w *AcmeWrapper) loadFile(path string) ([]byte, error) {
	//use custom load file callback?
	if w.Config.LoadFileCallback != nil {
		if b, err := w.Config.LoadFileCallback(path); err == nil {
			return b, nil
		} else if err != ErrNotHandled {
			return nil, err
		}
	}
	//default load from disk
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (w *AcmeWrapper) saveFile(path string, contents []byte) error {
	//use custom save file callback?
	if w.Config.SaveFileCallback != nil {
		if err := w.Config.SaveFileCallback(path, contents); err == nil {
			return nil
		} else if err != ErrNotHandled {
			return err
		}
	}
	//default save to disk (current user read+write only!)
	if err := ioutil.WriteFile(path, contents, 0600); err != nil {
		return err
	}
	return nil
}

