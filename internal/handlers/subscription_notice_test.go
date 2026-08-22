package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"subtrackr/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSubscription_CancellationNoticeDays(t *testing.T) {
	tests := []struct {
		name     string
		formBody string
		start    int
		want     int
	}{
		{
			name:     "sets notice period",
			formBody: "cancellation_notice_days=28",
			start:    0,
			want:     28,
		},
		{
			name:     "clears notice period with zero",
			formBody: "cancellation_notice_days=0",
			start:    28,
			want:     0,
		},
		{
			name:     "empty value clears notice period",
			formBody: "cancellation_notice_days=",
			start:    14,
			want:     0,
		},
		{
			name:     "negative value falls back to zero",
			formBody: "cancellation_notice_days=-5",
			start:    14,
			want:     0,
		},
		{
			name:     "value above cap is clamped to a year",
			formBody: "cancellation_notice_days=999",
			start:    0,
			want:     365,
		},
		{
			name:     "absent field leaves value unchanged",
			formBody: "name=Gym",
			start:    28,
			want:     28,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, svc := newSubscriptionUpdateTestRouter(t)

			created, err := svc.Create(&models.Subscription{
				Name:                   "Gym",
				Cost:                   40.00,
				Schedule:               "Annual",
				Status:                 "Active",
				ReminderEnabled:        true,
				CancellationNoticeDays: tc.start,
			})
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPost,
				"/subscriptions/"+strconv.FormatUint(uint64(created.ID), 10),
				strings.NewReader(tc.formBody),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

			updated, err := svc.GetByID(created.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.want, updated.CancellationNoticeDays)
		})
	}
}

// Changing the notice period moves the reminder anchor, so the renewal-reminder
// dedup state must reset; leaving it unchanged must preserve that state.
func TestUpdateSubscription_NoticeChangeResetsReminderState(t *testing.T) {
	tests := []struct {
		name      string
		formBody  string
		wantReset bool
	}{
		{name: "changed value resets dedup state", formBody: "cancellation_notice_days=14", wantReset: true},
		{name: "unchanged value preserves dedup state", formBody: "cancellation_notice_days=28", wantReset: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, svc := newSubscriptionUpdateTestRouter(t)

			sent := time.Now()
			renewal := time.Now().AddDate(0, 0, 40)
			created, err := svc.Create(&models.Subscription{
				Name:                   "Gym",
				Cost:                   40.00,
				Schedule:               "Annual",
				Status:                 "Active",
				ReminderEnabled:        true,
				RenewalDate:            &renewal,
				CancellationNoticeDays: 28,
			})
			require.NoError(t, err)

			created.LastReminderSent = &sent
			created.LastReminderRenewalDate = created.RenewalDate
			created.LastReminderWindow = 0
			_, err = svc.Update(created.ID, created)
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPost,
				"/subscriptions/"+strconv.FormatUint(uint64(created.ID), 10),
				strings.NewReader(tc.formBody),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

			updated, err := svc.GetByID(created.ID)
			require.NoError(t, err)
			if tc.wantReset {
				assert.Nil(t, updated.LastReminderSent)
				assert.Nil(t, updated.LastReminderRenewalDate)
				assert.Equal(t, -1, updated.LastReminderWindow)
			} else {
				assert.NotNil(t, updated.LastReminderSent)
				assert.Equal(t, 0, updated.LastReminderWindow)
			}
		})
	}
}
