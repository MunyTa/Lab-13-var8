package hr

import (
	"sort"
	"time"
)

type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Available bool   `json:"available"`
}

func GenerateTimeSlots(date time.Time, startHour, endHour, durationMinutes int) []TimeSlot {
	slots := []TimeSlot{}
	current := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, date.Location())
	end := time.Date(date.Year(), date.Month(), date.Day(), endHour, 0, 0, 0, date.Location())

	for current.Before(end) {
		slotEnd := current.Add(time.Duration(durationMinutes) * time.Minute)
		if slotEnd.After(end) {
			break
		}
		slots = append(slots, TimeSlot{
			Start: current,
			End:   slotEnd,
			Available: true,
		})
		current = slotEnd
	}

	return slots
}

func FindAvailableSlot(slots []TimeSlot, preferredTime time.Time, toleranceMinutes int) *TimeSlot {
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].Start.Before(slots[j].Start)
	})

	tolerance := time.Duration(toleranceMinutes) * time.Minute
	for _, slot := range slots {
		if slot.Available {
			diff := slot.Start.Sub(preferredTime)
			if diff < 0 {
				diff = -diff
			}
			if diff <= tolerance {
				return &slot
			}
		}
	}

	for _, slot := range slots {
		if slot.Available {
			return &slot
		}
	}

	return nil
}

func SelectInterviewers(requiredCount int, allInterviewers []string) []string {
	if requiredCount >= len(allInterviewers) {
		return allInterviewers
	}
	return allInterviewers[:requiredCount]
}
