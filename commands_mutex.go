package goexec

import "context"

type commandsMutex struct {
	highPriorityCommands   chan struct{}
	normalPriorityCommands chan struct{}
}

// newCommandsMutex creates a new commandsMutex with the specified channel capacities.
// Pass 0 for normalPriorityCapacity to disable concurrency control entirely.
func newCommandsMutex(highPriorityCapacity uint, normalPriorityCapacity uint) *commandsMutex {
	cm := &commandsMutex{}

	// If normal capacity is 0, no concurrency control at all
	if normalPriorityCapacity == 0 {
		return cm
	}

	if highPriorityCapacity != 0 {
		normalPriorityCapacity = highPriorityCapacity
	}
	// Create normal priority channel
	cm.normalPriorityCommands = make(chan struct{}, normalPriorityCapacity)

	// max value of 1,normal/10,input
	highPriorityCapacity = max(highPriorityCapacity, 1, normalPriorityCapacity/10)
	cm.highPriorityCommands = make(chan struct{}, highPriorityCapacity)

	return cm
}

// acquireSlot acquires a slot and returns the priority type that was actually acquired
func (c *commandsMutex) acquireSlot(ctx context.Context, highPriority bool) (acquiredHighPriority bool, err error) {
	if c == nil {
		return
	}
	// If both channels are nil, no locking needed
	if c.highPriorityCommands == nil && c.normalPriorityCommands == nil {
		return false, nil
	}

	if highPriority && c.highPriorityCommands != nil {
		select {
		case c.highPriorityCommands <- struct{}{}:
			return true, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	} else if c.normalPriorityCommands != nil {
		select {
		case c.normalPriorityCommands <- struct{}{}:
			return false, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	// No channels available to acquire
	return false, nil
}

func (c *commandsMutex) releaseSlot(acquiredHighPriority bool) {
	if c == nil {
		return
	}
	// If both channels are nil, no unlocking needed
	if c.highPriorityCommands == nil && c.normalPriorityCommands == nil {
		return
	}

	if acquiredHighPriority && c.highPriorityCommands != nil {
		<-c.highPriorityCommands
	} else if c.normalPriorityCommands != nil {
		<-c.normalPriorityCommands
	}
}
