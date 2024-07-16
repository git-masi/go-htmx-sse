package earnings

import "github.com/git-masi/go-htmx-sse/cmd/web/events"

func NewEarningPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

type PubSubEvent struct {
	EarningID int64
}
