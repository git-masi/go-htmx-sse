package workers

import "github.com/git-masi/paynext/cmd/internal-server/events"

const Topic = "Worker"

func NewWorkerPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

type PubSubEvent struct {
	WorkerID int64
	Event    Event
}
