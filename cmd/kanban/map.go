package main

import "unsafe"

// Map of arbitrary data to hold unorded state for `layout.List` items.
// This allows Gio programs to re-use a buffer of states for lists items in
// between frames. It is a grow-only buffer that expects entries to stabilise.
//
// This is designed along 2 constraints:
// 1. Performance
// 2. Type ambiguity
//
// Since Go doesn't have generics, I decided to give the caller type control
// by using `unsafe.Pointer`.
//
// The caller only has to ensure that the type they initialise it with is the
// type they attempt to cast out of it.
// Since the scope of use is small, this invariant is straighforward to uphold.
//
// Nonetheless, this style of API is primarily motived by re-use concerns when
// using common patterns in Gio (specifically `layout.List` state management).
// The static approach would be to copy-paste the same structures with different
// types everytime you have list state to manage.
//
// In light of Go generics incoming, this may become a moot issue. In the meantime
// this remains an experimental API that functions as expected.
type Map struct {
	data    map[string]unsafe.Pointer
	index   []string
	current int
}

// Begin prepares the map to be accessed.
// Require to reset iteration state each frame.
func (m *Map) Begin() {
	m.current = 0
	if m.data == nil {
		m.data = make(map[string]unsafe.Pointer)
	}
}

// New returns a value for the provided key.
// In the case no value exists, the initializer is used as the default value.
//
//	v := (*T)(m.New("foo", &T{}))
//
func (m *Map) New(k string, init unsafe.Pointer) unsafe.Pointer {
	if _, ok := m.data[k]; !ok {
		m.data[k] = init
		m.index = append(m.index, k)
	}
	return m.data[k]
}

// Next iterates over the collect, returning the data and a boolean to indicate
// the end.
//
// 	for v, ok := m.Next(); ok; v, ok = m.Next() {
//		t := (*T)(v)
// 	}
//
func (m *Map) Next() (unsafe.Pointer, bool) {
	if m.current >= len(m.index) {
		return nil, false
	}
	defer func() { m.current++ }()
	return m.data[m.index[m.current]], m.current <= len(m.index)-1
}
