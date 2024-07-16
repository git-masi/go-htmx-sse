package earnings

import "github.com/git-masi/paynext/cmd/web/events"

func NewEarningPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

type PubSubEvent struct {
	EarningID int64
}
