package workers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/git-masi/go-htmx-sse/cmd/web/events"
	"github.com/git-masi/go-htmx-sse/internal/.gen/model"
)

const SSE_PREFIX = "Worker"

func WorkerCreated(ctx context.Context, w model.Workers, payPeriodId int64) events.EventStreamFormat {
	out := new(bytes.Buffer)

	workerCreated(w, payPeriodId).Render(ctx, out)

	return events.EventStreamFormat{
		Event: fmt.Sprintf("%s%s", SSE_PREFIX, Created),
		Data:  out.String(),
	}
}
