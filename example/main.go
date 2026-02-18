package main

import (
	"fmt"
	"log"

	oblenergo "off-energy-mk-ua"
)

func main() {
	c := oblenergo.NewClient()

	testQueue := "6.1"

	// GetCurrentInfo
	fmt.Printf("=== GetCurrentInfo(%q) ===\n", testQueue)
	info, err := c.GetCurrentInfo(testQueue)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Queue: %s (id=%d)\n", info.Queue.Name, info.Queue.ID)
	fmt.Printf("  Status: %s (probably=%v)\n", info.Status, info.Probably)
	fmt.Printf("  Time slot: %s - %s\n", info.TimeSlot.Start, info.TimeSlot.End)

	// GetDailyInfo
	fmt.Printf("\n=== GetDailyInfo(%q) ===\n", testQueue)
	daily, err := c.GetDailyInfo(testQueue)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Queue: %s (id=%d)\n", daily.Queue.Name, daily.Queue.ID)
	for _, slot := range daily.Slots {
		marker := "  "
		if slot.Status == oblenergo.StatusOff {
			marker = "X "
			if slot.Probably {
				marker = "? "
			}
		}
		fmt.Printf("  %s %s - %s  %s\n", marker, slot.TimeSlot.Start, slot.TimeSlot.End, slot.Status)
	}

	// GetRemainingTime
	fmt.Printf("\n=== GetRemainingTime(%q) ===\n", testQueue)
	rem, err := c.GetRemainingTime(testQueue)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Queue: %s (id=%d)\n", rem.Queue.Name, rem.Queue.ID)
	fmt.Printf("  Status: %s (probably=%v)\n", rem.Status, rem.Probably)
	if rem.Status == oblenergo.StatusOn && rem.ShutoffAt != nil {
		fmt.Printf("  Shutoff in: %s (at %s)\n", rem.Remaining, rem.ShutoffAt.Format("15:04"))
		fmt.Printf("  Shutoff slot: %s - %s\n", rem.ShutoffSlot.Start, rem.ShutoffSlot.End)
	} else if rem.Status == oblenergo.StatusOff {
		fmt.Printf("  Power is OFF right now\n")
	} else {
		fmt.Printf("  No shutdown scheduled for today\n")
	}

	// GetTomorrowDailyInfo
	fmt.Printf("\n=== GetTomorrowDailyInfo(%q) ===\n", testQueue)
	tomorrow, err := c.GetTomorrowDailyInfo(testQueue)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Queue: %s (id=%d)\n", tomorrow.Queue.Name, tomorrow.Queue.ID)
	for _, slot := range tomorrow.Slots {
		marker := "  "
		if slot.Status == oblenergo.StatusOff {
			marker = "X "
			if slot.Probably {
				marker = "? "
			}
		}
		fmt.Printf("  %s %s - %s  %s\n", marker, slot.TimeSlot.Start, slot.TimeSlot.End, slot.Status)
	}
}
