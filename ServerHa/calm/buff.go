package calm

import (
	"errors"
	"ha/log"
	"io"
	"runtime"
	"sync"
)

type Buffer struct {
	reader io.Reader
	buf    []byte
	len    int
	lock   *sync.Mutex
}

func (b *Buffer) Len() int {
	return b.len
}

func (b *Buffer) ReadFromReader() (int, error) {

	buf := make([]byte, 65535)
	//n, err := b.reader.Read(b.buf[b.len:])
	n, err := b.reader.Read(buf)
	if err != nil {
		return n, err
	}
	log.Info("ReadFromReader:", n)
	//b.buf = append(b.buf[:0], b.buf[n:]...)
	b.lock.Lock()
	b.buf = append(b.buf[:b.len], buf[0:]...)
	b.len += n
	b.lock.Unlock()
	return n, nil
}

func (b *Buffer) Seek(n int) ([]byte, error) {
	if b.len >= n {
		tbuf := make([]byte, n)
		copy(tbuf, b.buf[:n])
		return tbuf, nil
	}
	return nil, errors.New("not enough seek")
}

func (b *Buffer) Read(n int) []byte {
	if b.len < n {
		return nil
	}
	tbuf := make([]byte, n)
	copy(tbuf, b.buf[:n])
	b.lock.Lock()
	b.buf = append(b.buf[:0], b.buf[n:]...)
	b.len -= n
	b.lock.Unlock()
	return tbuf
}

func (b *Buffer) Close() {
	b.buf = nil
	runtime.GC()
	return
}

func NewBuffer(reader io.Reader) Buffer {
	buf := make([]byte, 65535)
	return Buffer{reader, buf, 0, new(sync.Mutex)}
}

func DelBuffer(buff *Buffer) {
	buff.Close()
	return
}
