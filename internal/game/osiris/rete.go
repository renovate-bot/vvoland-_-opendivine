// SPDX-License-Identifier: GPL-3.0-only

package osiris

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrInvalidArity reports a tuple or memory with an invalid column count.
	ErrInvalidArity = errors.New("osiris: invalid tuple arity")
	// ErrInvalidJoinColumn reports a join key outside its input tuple.
	ErrInvalidJoinColumn = errors.New("osiris: invalid join column")
	// ErrUnknownEvent reports an assertion for an unregistered event handle.
	ErrUnknownEvent = errors.New("osiris: unknown event")
	// ErrDuplicateEvent reports a handle registered as an event more than once.
	ErrDuplicateEvent = errors.New("osiris: duplicate event")
)

// Tuple is one ordered row of typed Osiris values.
type Tuple []Value

type memoryEntry struct {
	tuple Tuple
	refs  int
}

type memoryListener interface {
	inserted(Tuple)
	removed(Tuple)
}

// Memory is a deduplicated RETE tuple memory with a fixed arity.
type Memory struct {
	arity     int
	entries   map[string]*memoryEntry
	listeners []memoryListener
}

// NewMemory creates an empty tuple memory.
func NewMemory(arity int) (*Memory, error) {
	if arity < 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidArity, arity)
	}
	return &Memory{arity: arity, entries: make(map[string]*memoryEntry)}, nil
}

// Arity returns the number of values in each tuple.
func (m *Memory) Arity() int {
	return m.arity
}

// Len returns the number of distinct tuples in memory.
func (m *Memory) Len() int {
	return len(m.entries)
}

// Tuples returns a deterministic snapshot of the distinct tuples in memory.
func (m *Memory) Tuples() []Tuple {
	keys := make([]string, 0, len(m.entries))
	for key := range m.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tuples := make([]Tuple, 0, len(keys))
	for _, key := range keys {
		tuples = append(tuples, cloneTuple(m.entries[key].tuple))
	}
	return tuples
}

// Insert adds a fact unless the same tuple is already present.
func (m *Memory) Insert(tuple Tuple) (bool, error) {
	if err := m.validate(tuple); err != nil {
		return false, err
	}
	return m.add(tuple, false), nil
}

// Remove retracts a fact if it is present.
func (m *Memory) Remove(tuple Tuple) (bool, error) {
	if err := m.validate(tuple); err != nil {
		return false, err
	}
	return m.subtract(tuple), nil
}

func (m *Memory) validate(tuple Tuple) error {
	if len(tuple) != m.arity {
		return fmt.Errorf("%w: have %d want %d", ErrInvalidArity, len(tuple), m.arity)
	}
	return nil
}

func (m *Memory) derive(tuple Tuple) {
	m.add(tuple, true)
}

func (m *Memory) underive(tuple Tuple) {
	if !m.subtract(tuple) {
		panic("osiris: retracting absent derived tuple")
	}
}

func (m *Memory) add(tuple Tuple, addReference bool) bool {
	key := tupleKey(tuple)
	if entry, ok := m.entries[key]; ok {
		if addReference {
			entry.refs++
		}
		return false
	}

	stored := cloneTuple(tuple)
	m.entries[key] = &memoryEntry{tuple: stored, refs: 1}
	for _, listener := range m.listeners {
		listener.inserted(stored)
	}
	return true
}

func (m *Memory) subtract(tuple Tuple) bool {
	key := tupleKey(tuple)
	entry, ok := m.entries[key]
	if !ok {
		return false
	}
	entry.refs--
	if entry.refs > 0 {
		return true
	}

	delete(m.entries, key)
	for _, listener := range m.listeners {
		listener.removed(entry.tuple)
	}
	return true
}

func (m *Memory) listen(listener memoryListener) {
	m.listeners = append(m.listeners, listener)
}

func tupleKey(tuple Tuple) string {
	buf := make([]byte, len(tuple)*4)
	for i, value := range tuple {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(value))
	}
	return string(buf)
}

func cloneTuple(tuple Tuple) Tuple {
	return append(Tuple(nil), tuple...)
}

// JoinKey identifies equal columns in a positive or negative join.
type JoinKey struct {
	Left  int
	Right int
}

func validateJoin(left, right *Memory, keys []JoinKey) error {
	if left == nil || right == nil {
		return errors.New("osiris: nil join memory")
	}
	for _, key := range keys {
		if key.Left < 0 || key.Left >= left.arity || key.Right < 0 || key.Right >= right.arity {
			return fmt.Errorf("%w: left %d/%d right %d/%d",
				ErrInvalidJoinColumn, key.Left, left.arity, key.Right, right.arity)
		}
	}
	return nil
}

func tuplesMatch(left, right Tuple, keys []JoinKey) bool {
	for _, key := range keys {
		if left[key.Left] != right[key.Right] {
			return false
		}
	}
	return true
}

func joinedTuple(left, right Tuple) Tuple {
	joined := make(Tuple, 0, len(left)+len(right))
	joined = append(joined, left...)
	return append(joined, right...)
}

type joinNode struct {
	left  *Memory
	right *Memory
	keys  []JoinKey
	out   *Memory
}

type joinInput struct {
	node *joinNode
	left bool
}

func (in joinInput) inserted(tuple Tuple) {
	if in.left {
		for _, right := range in.node.right.Tuples() {
			if tuplesMatch(tuple, right, in.node.keys) {
				in.node.out.derive(joinedTuple(tuple, right))
			}
		}
		return
	}
	for _, left := range in.node.left.Tuples() {
		if tuplesMatch(left, tuple, in.node.keys) {
			in.node.out.derive(joinedTuple(left, tuple))
		}
	}
}

func (in joinInput) removed(tuple Tuple) {
	if in.left {
		for _, right := range in.node.right.Tuples() {
			if tuplesMatch(tuple, right, in.node.keys) {
				in.node.out.underive(joinedTuple(tuple, right))
			}
		}
		return
	}
	for _, left := range in.node.left.Tuples() {
		if tuplesMatch(left, tuple, in.node.keys) {
			in.node.out.underive(joinedTuple(left, tuple))
		}
	}
}

// Join creates a memory containing concatenated tuples whose key columns
// compare equal.
func Join(left, right *Memory, keys ...JoinKey) (*Memory, error) {
	if err := validateJoin(left, right, keys); err != nil {
		return nil, err
	}
	out, err := NewMemory(left.arity + right.arity)
	if err != nil {
		return nil, err
	}
	node := &joinNode{left: left, right: right, keys: append([]JoinKey(nil), keys...), out: out}
	left.listen(joinInput{node: node, left: true})
	right.listen(joinInput{node: node})

	for _, leftTuple := range left.Tuples() {
		for _, rightTuple := range right.Tuples() {
			if tuplesMatch(leftTuple, rightTuple, node.keys) {
				out.derive(joinedTuple(leftTuple, rightTuple))
			}
		}
	}
	return out, nil
}

type negativeJoinNode struct {
	left    *Memory
	right   *Memory
	keys    []JoinKey
	out     *Memory
	matches map[string]int
}

type negativeJoinInput struct {
	node *negativeJoinNode
	left bool
}

func (in negativeJoinInput) inserted(tuple Tuple) {
	if in.left {
		count := 0
		for _, right := range in.node.right.Tuples() {
			if tuplesMatch(tuple, right, in.node.keys) {
				count++
			}
		}
		in.node.matches[tupleKey(tuple)] = count
		if count == 0 {
			in.node.out.derive(tuple)
		}
		return
	}

	for _, left := range in.node.left.Tuples() {
		if !tuplesMatch(left, tuple, in.node.keys) {
			continue
		}
		key := tupleKey(left)
		if in.node.matches[key] == 0 {
			in.node.out.underive(left)
		}
		in.node.matches[key]++
	}
}

func (in negativeJoinInput) removed(tuple Tuple) {
	if in.left {
		key := tupleKey(tuple)
		if in.node.matches[key] == 0 {
			in.node.out.underive(tuple)
		}
		delete(in.node.matches, key)
		return
	}

	for _, left := range in.node.left.Tuples() {
		if !tuplesMatch(left, tuple, in.node.keys) {
			continue
		}
		key := tupleKey(left)
		in.node.matches[key]--
		if in.node.matches[key] == 0 {
			in.node.out.derive(left)
		}
	}
}

// NegativeJoin creates a memory containing left tuples for which no matching
// right tuple exists.
func NegativeJoin(left, right *Memory, keys ...JoinKey) (*Memory, error) {
	if err := validateJoin(left, right, keys); err != nil {
		return nil, err
	}
	out, err := NewMemory(left.arity)
	if err != nil {
		return nil, err
	}
	node := &negativeJoinNode{
		left:    left,
		right:   right,
		keys:    append([]JoinKey(nil), keys...),
		out:     out,
		matches: make(map[string]int),
	}
	left.listen(negativeJoinInput{node: node, left: true})
	right.listen(negativeJoinInput{node: node})

	for _, leftTuple := range left.Tuples() {
		negativeJoinInput{node: node, left: true}.inserted(leftTuple)
	}
	return out, nil
}

type filterNode struct {
	predicate func(Tuple) bool
	out       *Memory
}

func (n *filterNode) inserted(tuple Tuple) {
	if n.predicate(tuple) {
		n.out.derive(tuple)
	}
}

func (n *filterNode) removed(tuple Tuple) {
	if n.predicate(tuple) {
		n.out.underive(tuple)
	}
}

// Filter creates a memory containing tuples accepted by predicate.
// The predicate must remain stable while a tuple is present in parent.
func Filter(parent *Memory, predicate func(Tuple) bool) (*Memory, error) {
	if parent == nil {
		return nil, errors.New("osiris: nil filter memory")
	}
	if predicate == nil {
		return nil, errors.New("osiris: nil filter predicate")
	}
	out, err := NewMemory(parent.arity)
	if err != nil {
		return nil, err
	}
	node := &filterNode{predicate: predicate, out: out}
	parent.listen(node)
	for _, tuple := range parent.Tuples() {
		node.inserted(tuple)
	}
	return out, nil
}

type actionNode struct {
	action func(Tuple)
}

func (n actionNode) inserted(tuple Tuple) {
	n.action(cloneTuple(tuple))
}

func (actionNode) removed(Tuple) {}

// OnMatch runs action whenever a new distinct tuple enters parent.
func OnMatch(parent *Memory, action func(Tuple)) error {
	if parent == nil {
		return errors.New("osiris: nil action memory")
	}
	if action == nil {
		return errors.New("osiris: nil action")
	}
	parent.listen(actionNode{action: action})
	return nil
}
