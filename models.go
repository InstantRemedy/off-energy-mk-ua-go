package oblenergo

import "time"

// OutageQueueType represents the type of outage queue.
type OutageQueueType int

const (
	QueueTypeCity     OutageQueueType = 1
	QueueTypeDistrict OutageQueueType = 2
	QueueTypeSub      OutageQueueType = 3
)

// OutageQueue represents a single outage queue entry.
type OutageQueue struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	TypeID    int        `json:"type_id"`
	Enabled   int        `json:"enabled"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	Deleted   int        `json:"deleted"`
}

func (q OutageQueue) IsEnabled() bool {
	return q.Enabled == 1
}

// TimeSeries represents a 30-minute time slot in the schedule.
type TimeSeries struct {
	ID        int        `json:"id"`
	Start     string     `json:"start"`
	End       string     `json:"end"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// OutageType represents the severity/certainty of an outage.
type OutageType string

const (
	OutageOff         OutageType = "OFF"
	OutageProbablyOff OutageType = "PROBABLY_OFF"
	OutageSureOff     OutageType = "SURE_OFF"
)

// ScheduleEntry represents a single outage event within an active schedule.
type ScheduleEntry struct {
	ID               int        `json:"id"`
	OutageScheduleID int        `json:"outage_schedule_id"`
	TimeSeriesID     int        `json:"time_series_id"`
	OutageQueueID    int        `json:"outage_queue_id"`
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
	Type             OutageType `json:"type"`
}

// ActiveSchedule represents a full day outage schedule with all its entries.
type ActiveSchedule struct {
	ID     int             `json:"id"`
	From   time.Time       `json:"from"`
	To     time.Time       `json:"to"`
	Series []ScheduleEntry `json:"series"`
}

// Status represents the power state for a queue.
type Status string

const (
	StatusOn  Status = "ON"
	StatusOff Status = "OFF"
)

// CurrentInfo holds the current power state for a queue.
type CurrentInfo struct {
	Queue    OutageQueue `json:"queue"`
	Status   Status      `json:"status"`
	Probably bool        `json:"probably"`
	TimeSlot TimeSeries  `json:"time_slot"`
}

// DailySlot holds status for a single 30-min slot.
type DailySlot struct {
	TimeSlot TimeSeries `json:"time_slot"`
	Status   Status     `json:"status"`
	Probably bool       `json:"probably"`
}

// DailyInfo holds the full day schedule for a queue.
type DailyInfo struct {
	Queue OutageQueue `json:"queue"`
	Slots []DailySlot `json:"slots"`
}

// RemainingTime holds the time remaining until power goes off.
type RemainingTime struct {
	Queue         OutageQueue   `json:"queue"`
	Status        Status        `json:"status"`
	Probably      bool          `json:"probably"`
	Remaining     time.Duration `json:"remaining"`
	ShutoffAt     *time.Time    `json:"shutoff_at,omitempty"`
	ShutoffSlot   *TimeSeries   `json:"shutoff_slot,omitempty"`
}
