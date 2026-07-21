// SPDX-License-Identifier: GPL-3.0-only

package osiris

import "fmt"

// Network owns the event entry points of a RETE graph.
type Network struct {
	events map[Handle]*Memory
}

// NewNetwork creates an empty RETE event registry.
func NewNetwork() *Network {
	return &Network{events: make(map[Handle]*Memory)}
}

// RegisterEvent creates the transient tuple memory for handle.
func (n *Network) RegisterEvent(handle Handle, arity int) (*Memory, error) {
	if _, exists := n.events[handle]; exists {
		return nil, fmt.Errorf("%w: %#x", ErrDuplicateEvent, uint32(handle))
	}
	memory, err := NewMemory(arity)
	if err != nil {
		return nil, err
	}
	n.events[handle] = memory
	return memory, nil
}

// AssertEvent inserts an event for one propagation cycle, then retracts it.
func (n *Network) AssertEvent(handle Handle, args Tuple) error {
	memory, ok := n.events[handle]
	if !ok {
		return fmt.Errorf("%w: %#x", ErrUnknownEvent, uint32(handle))
	}
	inserted, err := memory.Insert(args)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}
	_, err = memory.Remove(args)
	return err
}

// Event is one deferred gameplay event and its Osiris arguments.
type Event struct {
	Handle Handle
	Args   Tuple
}

// Queue batches gameplay events until the story update phase.
type Queue struct {
	events []Event
}

// Enqueue appends an event to the queue.
func (q *Queue) Enqueue(handle Handle, args Tuple) {
	q.events = append(q.events, Event{Handle: handle, Args: cloneTuple(args)})
}

// Len returns the number of pending events.
func (q *Queue) Len() int {
	return len(q.events)
}

// Flush asserts the current batch in order.
// Events enqueued by actions remain queued for the next flush.
func (q *Queue) Flush(network *Network) error {
	batch := q.events
	q.events = nil
	for i, event := range batch {
		if err := network.AssertEvent(event.Handle, event.Args); err != nil {
			q.events = append(batch[i:], q.events...)
			return err
		}
	}
	return nil
}
