package events

import (
	"bytes"
)

type EventStreamFormat struct {
	Event string
	// It might make sense for Data to be an interface like io.ByteReader
	// But string is easy enough for now
	Data string
}

func (e EventStreamFormat) Bytes() []byte {
	out := new(bytes.Buffer)

	out.WriteString("event: ")
	out.WriteString(e.Event)
	out.WriteString("\n")
	out.WriteString("data: ")
	out.WriteString(e.Data)
	out.WriteString("\n\n")

	return out.Bytes()
}
