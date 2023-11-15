package buffer

type stackBuffer struct {
	store []interface{}
	head  int
}

func (b *stackBuffer) add(element interface{}) bool {
	if b.head > len(b.store)-1 {
		return false
	} // buffer is full
	b.store[b.head] = element
	b.head++
	return true
}

func (b *stackBuffer) get() interface{} {
	if b.head == 0 {
		return nil
	}
	val := b.store[b.head-1]
	b.store = append(b.store[:b.head-1], b.store[b.head:]...)
	b.store = append(b.store, nil)
	b.head--
	return val
}
