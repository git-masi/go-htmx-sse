package earnings

import "github.com/git-masi/paynext/cmd/internal-server/events"

func NewEarningPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

type PubSubEvent struct {
	EarningID int64
}
