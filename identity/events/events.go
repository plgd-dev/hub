package events

import "google.golang.org/protobuf/proto"

func (e *Event) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *Event) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}
