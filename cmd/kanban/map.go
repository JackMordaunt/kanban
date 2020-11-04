package main

import "unsafe"

// Map of arbitrary data.
type Map struct {
	data map[string]unsafe.Pointer
}

func (m *Map) Begin() {
	if m.data == nil {
		m.data = make(map[string]unsafe.Pointer)
	}
}

func (m *Map) Next(k string, v unsafe.Pointer) unsafe.Pointer {
	if _, ok := m.data[k]; !ok {
		m.data[k] = v
	}
	return m.data[k]
}

func (m *Map) List() []unsafe.Pointer {
	list := []unsafe.Pointer{}
	for _, v := range m.data {
		if v != nil {
			list = append(list, v)
		}
	}
	return list
}
