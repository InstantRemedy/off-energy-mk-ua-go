package oblenergo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var kyivLocation *time.Location

func init() {
	var err error
	kyivLocation, err = time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic("failed to load Europe/Kyiv timezone: " + err.Error())
	}
}

const baseURL = "https://off.energy.mk.ua"

// Client communicates with the Mykolaiv Oblenergo outage API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) get(path string, result any) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// GetOutageQueues returns outage queues for the given type (1=city, 2=district, 3=sub).
func (c *Client) GetOutageQueues(queueType OutageQueueType) ([]OutageQueue, error) {
	var queues []OutageQueue
	err := c.get(fmt.Sprintf("/api/outage-queue/by-type/%d", queueType), &queues)
	return queues, err
}

func (c *Client) GetCityQueues() ([]OutageQueue, error) {
	return c.GetOutageQueues(QueueTypeCity)
}

func (c *Client) GetDistrictQueues() ([]OutageQueue, error) {
	return c.GetOutageQueues(QueueTypeDistrict)
}

func (c *Client) GetSubQueues() ([]OutageQueue, error) {
	return c.GetOutageQueues(QueueTypeSub)
}

// GetTimeSeries returns all 48 half-hour time slots.
func (c *Client) GetTimeSeries() ([]TimeSeries, error) {
	var series []TimeSeries
	err := c.get("/api/schedule/time-series", &series)
	return series, err
}

// GetActiveSchedule returns the currently active outage schedule.
func (c *Client) GetActiveSchedule() ([]ActiveSchedule, error) {
	var schedules []ActiveSchedule
	err := c.get("/api/v2/schedule/active", &schedules)
	return schedules, err
}

// findQueue searches all queue types for a queue with the given name.
func (c *Client) findQueue(name string) (*OutageQueue, error) {
	for _, qt := range []OutageQueueType{QueueTypeCity, QueueTypeDistrict, QueueTypeSub} {
		queues, err := c.GetOutageQueues(qt)
		if err != nil {
			return nil, err
		}
		for _, q := range queues {
			if q.Name == name {
				return &q, nil
			}
		}
	}
	return nil, fmt.Errorf("queue %q not found", name)
}

// currentTimeSeriesID returns the time series ID (1-48) for the current Kyiv time.
func currentTimeSeriesID(now time.Time) int {
	kyiv := now.In(kyivLocation)
	slot := kyiv.Hour()*2 + 1
	if kyiv.Minute() >= 30 {
		slot++
	}
	return slot
}

// parseSlotTime parses a time string like "08:30:00" into hour and minute.
func parseSlotTime(s string) (int, int) {
	var h, m, sec int
	fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec)
	return h, m
}

// slotToKyivTime converts a time series slot's start time to a time.Time on today in Kyiv.
func slotToKyivTime(slot TimeSeries, now time.Time) time.Time {
	kyiv := now.In(kyivLocation)
	h, m := parseSlotTime(slot.Start)
	return time.Date(kyiv.Year(), kyiv.Month(), kyiv.Day(), h, m, 0, 0, kyivLocation)
}

// activeScheduleForNow returns the schedule whose from/to range contains now.
func activeScheduleForNow(schedules []ActiveSchedule, now time.Time) *ActiveSchedule {
	for i := range schedules {
		if !now.Before(schedules[i].From) && now.Before(schedules[i].To) {
			return &schedules[i]
		}
	}
	// Fallback: return the last schedule if none match exactly
	if len(schedules) > 0 {
		return &schedules[len(schedules)-1]
	}
	return nil
}

// scheduleForTomorrow returns the schedule that covers tomorrow.
// Tomorrow's schedule has a "from" that is after current time.
func scheduleForTomorrow(schedules []ActiveSchedule, now time.Time) *ActiveSchedule {
	current := activeScheduleForNow(schedules, now)
	for i := range schedules {
		if current != nil && schedules[i].ID == current.ID {
			continue
		}
		if schedules[i].From.After(now) || schedules[i].From.Equal(now) {
			return &schedules[i]
		}
	}
	// If only one schedule, it might already be tomorrow's
	if len(schedules) > 0 {
		return &schedules[len(schedules)-1]
	}
	return nil
}

// entryStatus converts a schedule entry type to Status + probably flag.
func entryStatus(t OutageType) (Status, bool) {
	switch t {
	case OutageProbablyOff:
		return StatusOff, true
	case OutageOff, OutageSureOff:
		return StatusOff, false
	default:
		return StatusOff, false
	}
}

// GetCurrentInfo returns the current power state for a queue by name.
func (c *Client) GetCurrentInfo(name string) (*CurrentInfo, error) {
	queue, err := c.findQueue(name)
	if err != nil {
		return nil, err
	}

	timeSeries, err := c.GetTimeSeries()
	if err != nil {
		return nil, err
	}

	schedules, err := c.GetActiveSchedule()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	slotID := currentTimeSeriesID(now)

	var currentSlot TimeSeries
	for _, ts := range timeSeries {
		if ts.ID == slotID {
			currentSlot = ts
			break
		}
	}

	info := &CurrentInfo{
		Queue:    *queue,
		Status:   StatusOn,
		Probably: false,
		TimeSlot: currentSlot,
	}

	sched := activeScheduleForNow(schedules, now)
	if sched != nil {
		for _, entry := range sched.Series {
			if entry.OutageQueueID == queue.ID && entry.TimeSeriesID == slotID {
				info.Status, info.Probably = entryStatus(entry.Type)
				return info, nil
			}
		}
	}

	return info, nil
}

// GetDailyInfo returns the status for all 48 time slots for a queue by name.
func (c *Client) GetDailyInfo(name string) (*DailyInfo, error) {
	queue, err := c.findQueue(name)
	if err != nil {
		return nil, err
	}

	timeSeries, err := c.GetTimeSeries()
	if err != nil {
		return nil, err
	}

	schedules, err := c.GetActiveSchedule()
	if err != nil {
		return nil, err
	}

	// Build lookup from the current schedule only
	entryMap := map[int]ScheduleEntry{}
	now := time.Now()
	sched := activeScheduleForNow(schedules, now)
	if sched != nil {
		for _, entry := range sched.Series {
			if entry.OutageQueueID == queue.ID {
				entryMap[entry.TimeSeriesID] = entry
			}
		}
	}

	daily := &DailyInfo{
		Queue: *queue,
		Slots: make([]DailySlot, 0, len(timeSeries)),
	}

	for _, ts := range timeSeries {
		slot := DailySlot{
			TimeSlot: ts,
			Status:   StatusOn,
			Probably: false,
		}
		if entry, ok := entryMap[ts.ID]; ok {
			slot.Status, slot.Probably = entryStatus(entry.Type)
		}
		daily.Slots = append(daily.Slots, slot)
	}

	return daily, nil
}

// GetRemainingTime returns how long until power goes off for a queue.
// If power is already off, Remaining is 0.
func (c *Client) GetRemainingTime(name string) (*RemainingTime, error) {
	queue, err := c.findQueue(name)
	if err != nil {
		return nil, err
	}

	timeSeries, err := c.GetTimeSeries()
	if err != nil {
		return nil, err
	}

	schedules, err := c.GetActiveSchedule()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	currentSlotID := currentTimeSeriesID(now)

	// Build lookup from the current schedule only
	entryMap := map[int]ScheduleEntry{}
	sched := activeScheduleForNow(schedules, now)
	if sched != nil {
		for _, entry := range sched.Series {
			if entry.OutageQueueID == queue.ID {
				entryMap[entry.TimeSeriesID] = entry
			}
		}
	}

	result := &RemainingTime{
		Queue:  *queue,
		Status: StatusOn,
	}

	// Check if currently off
	if entry, ok := entryMap[currentSlotID]; ok {
		result.Status, result.Probably = entryStatus(entry.Type)
		result.Remaining = 0
		return result, nil
	}

	// Currently on — find next off slot starting from current+1
	tsMap := map[int]TimeSeries{}
	for _, ts := range timeSeries {
		tsMap[ts.ID] = ts
	}

	for i := currentSlotID + 1; i <= 48; i++ {
		if _, ok := entryMap[i]; ok {
			slot := tsMap[i]
			shutoffTime := slotToKyivTime(slot, now)
			result.Remaining = shutoffTime.Sub(now.In(kyivLocation))
			result.ShutoffAt = &shutoffTime
			result.ShutoffSlot = &slot
			return result, nil
		}
	}

	// No shutdown found for the rest of the day
	result.Remaining = 0
	return result, nil
}

// GetTomorrowDailyInfo returns the status for all 48 time slots for tomorrow.
func (c *Client) GetTomorrowDailyInfo(name string) (*DailyInfo, error) {
	queue, err := c.findQueue(name)
	if err != nil {
		return nil, err
	}

	timeSeries, err := c.GetTimeSeries()
	if err != nil {
		return nil, err
	}

	schedules, err := c.GetActiveSchedule()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sched := scheduleForTomorrow(schedules, now)

	entryMap := map[int]ScheduleEntry{}
	if sched != nil {
		for _, entry := range sched.Series {
			if entry.OutageQueueID == queue.ID {
				entryMap[entry.TimeSeriesID] = entry
			}
		}
	}

	daily := &DailyInfo{
		Queue: *queue,
		Slots: make([]DailySlot, 0, len(timeSeries)),
	}

	for _, ts := range timeSeries {
		slot := DailySlot{
			TimeSlot: ts,
			Status:   StatusOn,
			Probably: false,
		}
		if entry, ok := entryMap[ts.ID]; ok {
			slot.Status, slot.Probably = entryStatus(entry.Type)
		}
		daily.Slots = append(daily.Slots, slot)
	}

	return daily, nil
}
