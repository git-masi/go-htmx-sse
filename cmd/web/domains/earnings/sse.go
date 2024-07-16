package earnings

import (
	"bytes"
	"context"
	"fmt"

	"github.com/git-masi/go-htmx-sse/cmd/web/events"
	"github.com/git-masi/go-htmx-sse/internal/.gen/model"
)

const SSE_PREFIX = "Earning"

func EarningCreated(ctx context.Context, e model.Earnings) events.EventStreamFormat {
	out := new(bytes.Buffer)

	earningCreated(e).Render(ctx, out)

	return events.EventStreamFormat{
		Event: fmt.Sprintf("%s%s", SSE_PREFIX, Created),
		Data:  out.String(),
	}
}
