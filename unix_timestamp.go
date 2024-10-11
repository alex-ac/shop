package shop

import (
	"encoding/json"
	"time"
)

type UnixTimestamp struct {
	time.Time
}

func (ut UnixTimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(ut.Time.Unix())
}

func (ut *UnixTimestamp) UnmarshalJSON(d []byte) (err error) {
	var v int64
	err = json.Unmarshal(d, &v)
	if err == nil {
		ut.Time = time.Unix(v, 0)
	}
	return
}
