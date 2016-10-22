package tbus

type pfxMap struct {
	value interface{}
	nodes map[uint8]*pfxMap
}

func newPfxMap() *pfxMap {
	return &pfxMap{nodes: make(map[uint8]*pfxMap)}
}

func (m *pfxMap) lookup(keys []uint8) interface{} {
	node := m
	for {
		if len(keys) == 0 {
			return node.value
		}
		if node = node.nodes[keys[0]]; node == nil {
			return nil
		}
		keys = keys[1:]
	}
}

func (m *pfxMap) insert(keys []uint8, val interface{}) (old interface{}) {
	node := m
	for {
		if len(keys) == 0 {
			break
		}
		next := node.nodes[keys[0]]
		if next == nil {
			next = newPfxMap()
			node.nodes[keys[0]] = next
		}
		node = next
		keys = keys[1:]
	}

	if old = node.value; old == nil {
		node.value = val
	}
	return
}

func (m *pfxMap) remove(keys []uint8) {
	if len(keys) == 0 {
		return
	}
	if node := m.nodes[keys[0]]; node != nil {
		node.remove(keys[1:])
		if node.empty() {
			delete(m.nodes, keys[0])
		}
	}
}

func (m *pfxMap) empty() bool {
	return len(m.nodes) == 0 && m.value == nil
}
