package workers

import "github.com/git-masi/paynext/cmd/internal-server/events"

func NewWorkerPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

// The topic will be something like `CREATED`
type PubSubEvent struct {
	WorkerID int64
}
