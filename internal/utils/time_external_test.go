package utils_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/git-masi/go-htmx-sse/internal/utils"
)

func TestGetWeekStartEnd(t *testing.T) {
	tests := []struct {
		now       time.Time
		startDate string
		endDate   string
	}{
		{now: time.Unix(1703980800, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704067200, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704153600, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704240000, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704326400, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704412800, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704499200, 0).UTC(), startDate: "Sun Dec 31", endDate: "Sat Jan 06"},
		{now: time.Unix(1704585600, 0).UTC(), startDate: "Sun Jan 07", endDate: "Sat Jan 13"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("given: %d, it should return: %s, %s", tc.now.Unix(), tc.startDate, tc.endDate), func(t *testing.T) {
			startDate, endDate := utils.GetWeekStartEnd(tc.now)

			if !strings.Contains(startDate.Format(time.RubyDate), tc.startDate) {
				t.Errorf("got start date: %s, want %s", startDate.Format(time.RubyDate), tc.startDate)
			}

			if !strings.Contains(endDate.Format(time.RubyDate), tc.endDate) {
				t.Errorf("got end date: %s, want %s", endDate.Format(time.RubyDate), tc.endDate)
			}
		})
	}
}
