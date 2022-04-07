package nebula

import (
	"time"

	"github.com/slackhq/nebula/firewall"
)

// How many timer objects should be cached
const timerCacheMax = 50000

var emptyFWPacket = firewall.Packet{}

type TimerWheel struct {
	// Current tick
	current int

	// Cheat on finding the length of the wheel
	wheelLen int

	// Last time we ticked, since we are lazy ticking
	lastTick *time.Time

	// Durations of a tick and the entire wheel
	tickDuration  time.Duration
	wheelDuration time.Duration

	// The actual wheel which is just a set of singly linked lists, head/tail pointers
	wheel []*TimeoutList

	// Singly linked list of items that have timed out of the wheel
	expired *TimeoutList

	// Item cache to avoid garbage collect
	itemCache   *TimeoutItem
	itemsCached int
}

// Represents a tick in the wheel
type TimeoutList struct {
	Head *TimeoutItem
	Tail *TimeoutItem
}

// Represents an item within a tick
type TimeoutItem struct {
	Packet firewall.Packet
	Next   *TimeoutItem
}

// Builds a timer wheel and identifies the tick duration and wheel duration from the provided values
// Purge must be called once per entry to actually remove anything
func NewTimerWheel(min, max time.Duration) *TimerWheel {
	//TODO provide an error
	//if min >= max {
	//	return nil
	//}

	// Round down and add 1 so we can have the smallest # of ticks in the wheel and still account for a full
	// max duration
	wLen := int((max / min) + 1)

	tw := TimerWheel{
		wheelLen:      wLen,
		wheel:         make([]*TimeoutList, wLen),
		tickDuration:  min,
		wheelDuration: max,
		expired:       &TimeoutList{},
	}

	for i := range tw.wheel {
		tw.wheel[i] = &TimeoutList{}
	}

	return &tw
}

// Add will add a firewall.Packet to the wheel in it's proper timeout
func (tw *TimerWheel) Add(v firewall.Packet, timeout time.Duration) *TimeoutItem {
	// Check and see if we should progress the tick
	tw.advance(time.Now())

	i := tw.findWheel(timeout)

	// Try to fetch off the cache
	ti := tw.itemCache
	if ti != nil {
		tw.itemCache = ti.Next
		tw.itemsCached--
		ti.Next = nil
	} else {
		ti = &TimeoutItem{}
	}

	// Relink and return
	ti.Packet = v
	if tw.wheel[i].Tail == nil {
		tw.wheel[i].Head = ti
		tw.wheel[i].Tail = ti
	} else {
		tw.wheel[i].Tail.Next = ti
		tw.wheel[i].Tail = ti
	}

	return ti
}

func (tw *TimerWheel) Purge() (firewall.Packet, bool) {
	if tw.expired.Head == nil {
		return emptyFWPacket, false
	}

	ti := tw.expired.Head
	tw.expired.Head = ti.Next

	if tw.expired.Head == nil {
		tw.expired.Tail = nil
	}

	// Clear out the items references
	ti.Next = nil

	// Maybe cache it for later
	if tw.itemsCached < timerCacheMax {
		ti.Next = tw.itemCache
		tw.itemCache = ti
		tw.itemsCached++
	}

	return ti.Packet, true
}

// advance will move the wheel forward by proper number of ticks. The caller _should_ lock the wheel before calling this
func (tw *TimerWheel) findWheel(timeout time.Duration) (i int) {
	if timeout < tw.tickDuration {
		// Can't track anything below the set resolution
		timeout = tw.tickDuration
	} else if timeout > tw.wheelDuration {
		// We aren't handling timeouts greater than the wheels duration
		timeout = tw.wheelDuration
	}

	// Find the next highest, rounding up
	tick := int(((timeout - 1) / tw.tickDuration) + 1)

	// Add another tick since the current tick may almost be over then map it to the wheel from our
	// current position
	tick += tw.current + 1
	if tick >= tw.wheelLen {
		tick -= tw.wheelLen
	}

	return tick
}

// advance will lock and move the wheel forward by proper number of ticks.
func (tw *TimerWheel) advance(now time.Time) {
	if tw.lastTick == nil {
		tw.lastTick = &now
	}

	// We want to round down
	ticks := int(now.Sub(*tw.lastTick) / tw.tickDuration)
	adv := ticks
	if ticks > tw.wheelLen {
		ticks = tw.wheelLen
	}

	for i := 0; i < ticks; i++ {
		tw.current++
		if tw.current >= tw.wheelLen {
			tw.current = 0
		}

		if tw.wheel[tw.current].Head != nil {
			// We need to append the expired items as to not starve evicting the oldest ones
			if tw.expired.Tail == nil {
				tw.expired.Head = tw.wheel[tw.current].Head
				tw.expired.Tail = tw.wheel[tw.current].Tail
			} else {
				tw.expired.Tail.Next = tw.wheel[tw.current].Head
				tw.expired.Tail = tw.wheel[tw.current].Tail
			}

			tw.wheel[tw.current].Head = nil
			tw.wheel[tw.current].Tail = nil
		}
	}

	// Advance the tick based on duration to avoid losing some accuracy
	newTick := tw.lastTick.Add(tw.tickDuration * time.Duration(adv))
	tw.lastTick = &newTick
}
