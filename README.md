# off-energy-mk-ua

Go library for communicating with the [Mykolaiv Oblenergo](https://off.energy.mk.ua/) power outage API.

## Installation

```bash
go get off-energy-mk-ua
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	oblenergo "off-energy-mk-ua"
)

func main() {
	c := oblenergo.NewClient()

	// Check current power status for queue 1.1
	info, err := c.GetCurrentInfo("1.1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status: %s (probably=%v)\n", info.Status, info.Probably)
}
```

## API Reference

### Client

```go
c := oblenergo.NewClient()
```

Creates a new client with default settings (base URL: `https://off.energy.mk.ua`, timeout: 15s).

### Low-level methods

| Method | Description |
|---|---|
| `GetOutageQueues(queueType)` | Get outage queues by type (1=city, 2=district, 3=sub) |
| `GetCityQueues()` | Get city queues (type 1) |
| `GetDistrictQueues()` | Get district queues (type 2) |
| `GetSubQueues()` | Get sub-queues (type 3) |
| `GetTimeSeries()` | Get all 48 half-hour time slots |
| `GetActiveSchedule()` | Get raw active outage schedules |

### High-level methods

#### GetCurrentInfo(name string) (*CurrentInfo, error)

Returns the current power status for a queue by name. Supports all queue types.

```go
info, _ := c.GetCurrentInfo("1.1")
// info.Status   — oblenergo.StatusOn or oblenergo.StatusOff
// info.Probably — true if status is uncertain (PROBABLY_OFF)
// info.TimeSlot — current 30-min time slot
// info.Queue    — queue details
```

#### GetDailyInfo(name string) (*DailyInfo, error)

Returns the status for all 48 time slots today.

```go
daily, _ := c.GetDailyInfo("1.1")
for _, slot := range daily.Slots {
    fmt.Printf("%s-%s  %s (probably=%v)\n",
        slot.TimeSlot.Start, slot.TimeSlot.End,
        slot.Status, slot.Probably)
}
```

#### GetRemainingTime(name string) (*RemainingTime, error)

Returns the remaining time until power goes off. If power is already off, `Remaining` is 0.

```go
rem, _ := c.GetRemainingTime("1.1")
if rem.Status == oblenergo.StatusOn && rem.ShutoffAt != nil {
    fmt.Printf("Power off in %s\n", rem.Remaining)
} else if rem.Status == oblenergo.StatusOff {
    fmt.Println("Power is OFF")
}
```

#### GetTomorrowDailyInfo(name string) (*DailyInfo, error)

Same as `GetDailyInfo` but for tomorrow's schedule.

```go
tomorrow, _ := c.GetTomorrowDailyInfo("1.1")
```

### Queue names

| Type | Names |
|---|---|
| City (type 1) | `1`, `2`, ... `10`, `1(P)`, `2(P)`, ... `10(P)` |
| District (type 2) | `1`, `2`, `3` |
| Sub-queue (type 3) | `1.1`, `1.2`, `2.1`, `2.2`, ... `6.1`, `6.2` |

### Status types

| Status | Probably | Meaning |
|---|---|---|
| `ON` | `false` | Power is on |
| `OFF` | `false` | Power is off (certain) |
| `OFF` | `true` | Power is probably off (uncertain, from `PROBABLY_OFF`) |

All times are calculated in `Europe/Kyiv` timezone.

## Run example

```bash
go run ./example/
```

## License

GPLv3
