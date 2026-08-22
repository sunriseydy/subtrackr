package service

import (
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNoticeReminderService(t *testing.T) *SubscriptionService {
	t.Helper()
	db := setupRenewalReminderTestDB(t)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	categoryService := NewCategoryService(repository.NewCategoryRepository(db))
	return NewSubscriptionService(subscriptionRepo, categoryService)
}

// A subscription with a cancellation notice period must be reminded relative to
// its cancel-by date (renewal date minus notice), not the renewal date itself.
func TestGetSubscriptionsNeedingReminders_NoticePeriodAnchorsToCancelBy(t *testing.T) {
	svc := newNoticeReminderService(t)
	now := time.Now()

	// Renews in 30 days with 28 days notice: cancel-by is ~2 days away, so it
	// must fire even though the renewal date is far outside the 7-day window.
	_, err := svc.Create(&models.Subscription{
		Name:                   "Gym Contract",
		Cost:                   40.00,
		Schedule:               "Annual",
		Status:                 "Active",
		ReminderEnabled:        true,
		RenewalDate:            timePtr(now.AddDate(0, 0, 30)),
		CancellationNoticeDays: 28,
	})
	require.NoError(t, err)

	// Same renewal distance without notice: outside the window, must not fire.
	_, err = svc.Create(&models.Subscription{
		Name:            "No Notice",
		Cost:            10.00,
		Schedule:        "Monthly",
		Status:          "Active",
		ReminderEnabled: true,
		RenewalDate:     timePtr(now.AddDate(0, 0, 30)),
	})
	require.NoError(t, err)

	result, err := svc.GetSubscriptionsNeedingReminders([]int{7, 3, 0})
	require.NoError(t, err)

	require.Len(t, result, 1)
	for sub, daysUntil := range result {
		assert.Equal(t, "Gym Contract", sub.Name)
		assert.LessOrEqual(t, daysUntil, 2, "daysUntil must be measured against the cancel-by date")
	}
}

// Once the cancel-by deadline has passed, reminders fall back to the renewal
// date so the user still hears about the upcoming charge. This also covers a
// notice period >= the billing cycle, whose anchor would otherwise be
// permanently in the past and silently disable all reminders.
func TestGetSubscriptionsNeedingReminders_PassedCancelByFallsBackToRenewal(t *testing.T) {
	svc := newNoticeReminderService(t)
	now := time.Now()

	_, err := svc.Create(&models.Subscription{
		Name:                   "Missed Deadline",
		Cost:                   40.00,
		Schedule:               "Annual",
		Status:                 "Active",
		ReminderEnabled:        true,
		RenewalDate:            timePtr(now.AddDate(0, 0, 5)),
		CancellationNoticeDays: 20,
	})
	require.NoError(t, err)

	result, err := svc.GetSubscriptionsNeedingReminders([]int{7, 3, 0})
	require.NoError(t, err)
	require.Len(t, result, 1)
	for sub, daysUntil := range result {
		assert.Equal(t, "Missed Deadline", sub.Name)
		assert.GreaterOrEqual(t, daysUntil, 4, "daysUntil must fall back to the renewal date, not the passed deadline")
	}
}

// A notice period longer than a year is clamped on write everywhere, including
// paths that bypass the HTML-form parser (MCP, restore, direct service calls).
func TestCancellationNoticeDaysClampedOnWrite(t *testing.T) {
	svc := newNoticeReminderService(t)

	created, err := svc.Create(&models.Subscription{
		Name:                   "Overlong Notice",
		Cost:                   10.00,
		Schedule:               "Annual",
		Status:                 "Active",
		CancellationNoticeDays: 100000,
	})
	require.NoError(t, err)
	assert.Equal(t, 365, created.CancellationNoticeDays)

	created.CancellationNoticeDays = -5
	updated, err := svc.Update(created.ID, created)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.CancellationNoticeDays)
}

// The de-duplication state written after a send must suppress a second fire for
// the same renewal date when the anchor is the cancel-by date.
func TestGetSubscriptionsNeedingReminders_NoticePeriodDeduplicates(t *testing.T) {
	svc := newNoticeReminderService(t)
	now := time.Now()

	created, err := svc.Create(&models.Subscription{
		Name:                   "Gym Contract",
		Cost:                   40.00,
		Schedule:               "Annual",
		Status:                 "Active",
		ReminderEnabled:        true,
		RenewalDate:            timePtr(now.AddDate(0, 0, 30)),
		CancellationNoticeDays: 28,
	})
	require.NoError(t, err)

	result, err := svc.GetSubscriptionsNeedingReminders([]int{7, 3, 0})
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Simulate the scheduler recording the send, exactly as main.go does.
	for sub, daysUntil := range result {
		sent := time.Now()
		sub.LastReminderSent = &sent
		renewalCopy := *sub.RenewalDate
		sub.LastReminderRenewalDate = &renewalCopy
		sub.LastReminderWindow = daysUntil
		_, err := svc.Update(sub.ID, sub)
		require.NoError(t, err)
	}

	again, err := svc.GetSubscriptionsNeedingReminders([]int{7, 3, 0})
	require.NoError(t, err)
	assert.Empty(t, again, "already-sent reminder must not fire twice for the same renewal date")

	// Sanity: the subscription still exists and kept its notice period.
	reloaded, err := svc.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, 28, reloaded.CancellationNoticeDays)
}

func TestCancelByDate(t *testing.T) {
	renewal := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)

	withNotice := models.Subscription{RenewalDate: &renewal, CancellationNoticeDays: 28}
	require.NotNil(t, withNotice.CancelByDate())
	assert.Equal(t, time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), *withNotice.CancelByDate())
	assert.Equal(t, *withNotice.CancelByDate(), *withNotice.ReminderAnchorDate())

	noNotice := models.Subscription{RenewalDate: &renewal}
	assert.Nil(t, noNotice.CancelByDate())
	require.NotNil(t, noNotice.ReminderAnchorDate())
	assert.Equal(t, renewal, *noNotice.ReminderAnchorDate())

	noRenewal := models.Subscription{CancellationNoticeDays: 14}
	assert.Nil(t, noRenewal.CancelByDate())
	assert.Nil(t, noRenewal.ReminderAnchorDate())

	// A passed deadline is still reported by CancelByDate (for display) but not
	// by UpcomingCancelByDate, and the reminder anchor reverts to the renewal date.
	soon := time.Now().AddDate(0, 0, 5)
	missed := models.Subscription{RenewalDate: &soon, CancellationNoticeDays: 20}
	require.NotNil(t, missed.CancelByDate())
	assert.Nil(t, missed.UpcomingCancelByDate())
	require.NotNil(t, missed.ReminderAnchorDate())
	assert.Equal(t, soon, *missed.ReminderAnchorDate())
}
