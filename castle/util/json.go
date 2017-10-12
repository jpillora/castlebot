package util

import (
	"encoding/json"
	"regexp"
	"time"
)

// Duration is a wrapper around time.Duration which implements json.Unmarshaler and json.Marshaler.
// It marshals and unmarshals the duration as a string in the format accepted by time.ParseDuration and returned by time.Duration.String.
type Duration time.Duration

func (d Duration) D() time.Duration {
	return time.Duration(d)
}

// MarshalJSON implements the json.Marshaler interface. The duration is a quoted-string in the format accepted by time.ParseDuration and returned by time.Duration.String.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.D().String() + `"`), nil
}

var intRe = regexp.MustCompile(`^\d+$`)

// UnmarshalJSON implements the json.Unmarshaler interface. The duration is expected to be a quoted-string of a duration in the format accepted by time.ParseDuration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	//support old int seconds data
	if intRe.Match(data) {
		//convert to "<int>s"
		data = []byte(`"` + string(data) + `s"`)
	}
	//
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tmp, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(tmp)
	return nil
}
