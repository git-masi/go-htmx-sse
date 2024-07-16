package workers

import "github.com/git-masi/go-htmx-sse/cmd/web/events"

const Topic = "Worker"

func NewWorkerPubSub() *events.PubSub[PubSubEvent] {
	return events.NewPubSub[PubSubEvent]()
}

type PubSubEvent struct {
	WorkerID int64
	Event    Event
}
