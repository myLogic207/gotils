package buffer

type queueBuffer struct {
	store []interface{}
	head  int
}

func (b *queueBuffer) add(element interface{}) bool {
	if len(b.store) == b.head {
		return false
	}
	b.store[b.head] = element
	b.head++
	return true
}

func (b *queueBuffer) get() interface{} {
	val := b.store[0]
	b.store = b.store[1:]
	b.store = append(b.store, nil)
	b.head--
	return val
}
