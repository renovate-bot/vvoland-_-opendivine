// SPDX-License-Identifier: GPL-3.0-only

package osiris

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func testValue(t *testing.T, kind Type, payload uint32) Value {
	t.Helper()
	value, err := NewValue(kind, payload)
	assert.NilError(t, err)
	return value
}

func testMemory(t *testing.T, arity int) *Memory {
	t.Helper()
	memory, err := NewMemory(arity)
	assert.NilError(t, err)
	return memory
}

func TestValueEncoding(t *testing.T) {
	t.Parallel()

	value := testValue(t, TypeObject, 0x12345)
	assert.Equal(t, value.Type(), TypeObject)
	assert.Equal(t, value.Payload(), uint32(0x12345))
}

func TestJoinEventWithFact(t *testing.T) {
	t.Parallel()

	network := NewNetwork()
	handle := testValue(t, TypeFunction, 1)
	events, err := network.RegisterEvent(handle, 1)
	assert.NilError(t, err)
	facts := testMemory(t, 1)
	matches, err := Join(events, facts, JoinKey{Left: 0, Right: 0})
	assert.NilError(t, err)

	object := testValue(t, TypeObject, 42)
	other := testValue(t, TypeObject, 43)
	inserted, err := facts.Insert(Tuple{object})
	assert.NilError(t, err)
	assert.Equal(t, inserted, true)

	var fired []Tuple
	assert.NilError(t, OnMatch(matches, func(tuple Tuple) {
		fired = append(fired, tuple)
	}))

	assert.NilError(t, network.AssertEvent(handle, Tuple{other}))
	assert.NilError(t, network.AssertEvent(handle, Tuple{object}))
	assert.Equal(t, len(fired), 1)
	assert.DeepEqual(t, fired[0], Tuple{object, object})
	assert.Equal(t, events.Len(), 0)
	assert.Equal(t, matches.Len(), 0)
}

func TestMemoryDeduplicatesFacts(t *testing.T) {
	t.Parallel()

	memory := testMemory(t, 1)
	fact := Tuple{testValue(t, TypeInteger, 7)}
	inserted, err := memory.Insert(fact)
	assert.NilError(t, err)
	assert.Equal(t, inserted, true)
	inserted, err = memory.Insert(fact)
	assert.NilError(t, err)
	assert.Equal(t, inserted, false)
	assert.Equal(t, memory.Len(), 1)
}

func TestJoinRetractsMatches(t *testing.T) {
	t.Parallel()

	left := testMemory(t, 1)
	right := testMemory(t, 1)
	matches, err := Join(left, right, JoinKey{Left: 0, Right: 0})
	assert.NilError(t, err)
	value := testValue(t, TypeInteger, 3)

	_, err = left.Insert(Tuple{value})
	assert.NilError(t, err)
	_, err = right.Insert(Tuple{value})
	assert.NilError(t, err)
	assert.Equal(t, matches.Len(), 1)

	removed, err := right.Remove(Tuple{value})
	assert.NilError(t, err)
	assert.Equal(t, removed, true)
	assert.Equal(t, matches.Len(), 0)
}

func TestNegativeJoinReactsToRightMemory(t *testing.T) {
	t.Parallel()

	eligible := testMemory(t, 1)
	completed := testMemory(t, 1)
	matches, err := NegativeJoin(eligible, completed, JoinKey{Left: 0, Right: 0})
	assert.NilError(t, err)
	object := testValue(t, TypeObject, 9)

	_, err = eligible.Insert(Tuple{object})
	assert.NilError(t, err)
	assert.Equal(t, matches.Len(), 1)

	_, err = completed.Insert(Tuple{object})
	assert.NilError(t, err)
	assert.Equal(t, matches.Len(), 0)

	_, err = completed.Remove(Tuple{object})
	assert.NilError(t, err)
	assert.Equal(t, matches.Len(), 1)

	_, err = eligible.Remove(Tuple{object})
	assert.NilError(t, err)
	assert.Equal(t, matches.Len(), 0)
}

func TestFilterPropagatesAcceptedTuples(t *testing.T) {
	t.Parallel()

	parent := testMemory(t, 1)
	filtered, err := Filter(parent, func(tuple Tuple) bool {
		return tuple[0].Payload() >= 10
	})
	assert.NilError(t, err)

	_, err = parent.Insert(Tuple{testValue(t, TypeInteger, 9)})
	assert.NilError(t, err)
	_, err = parent.Insert(Tuple{testValue(t, TypeInteger, 10)})
	assert.NilError(t, err)
	assert.Equal(t, filtered.Len(), 1)
}

func TestQueueFlushesEventsInOrder(t *testing.T) {
	t.Parallel()

	network := NewNetwork()
	handle := testValue(t, TypeFunction, 2)
	events, err := network.RegisterEvent(handle, 1)
	assert.NilError(t, err)

	var got []uint32
	assert.NilError(t, OnMatch(events, func(tuple Tuple) {
		got = append(got, tuple[0].Payload())
	}))

	var queue Queue
	queue.Enqueue(handle, Tuple{testValue(t, TypeInteger, 3)})
	queue.Enqueue(handle, Tuple{testValue(t, TypeInteger, 1)})
	queue.Enqueue(handle, Tuple{testValue(t, TypeInteger, 2)})
	assert.Equal(t, queue.Len(), 3)
	assert.NilError(t, queue.Flush(network))
	assert.Equal(t, queue.Len(), 0)
	assert.Assert(t, cmp.DeepEqual(got, []uint32{3, 1, 2}))
}
