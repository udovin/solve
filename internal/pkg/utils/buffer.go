package utils

import (
	"strings"
	"sync"
)

type TruncateBuffer struct {
	buffer strings.Builder
	limit  int
	mutex  sync.Mutex
}

func NewTruncateBuffer(limit int) *TruncateBuffer {
	return &TruncateBuffer{limit: 2048}
}

func (b *TruncateBuffer) String() string {
	return fixUTF8String(b.buffer.String())
}

func (b *TruncateBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	l := len(p)
	if b.buffer.Len()+l > b.limit {
		p = p[:b.limit-b.buffer.Len()]
	}
	if len(p) == 0 {
		return l, nil
	}
	n, err := b.buffer.Write(p)
	if err != nil {
		return n, err
	}
	return l, nil
}

func fixUTF8String(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\u0000", "")
}
