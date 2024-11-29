package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
)

var ctxes = []context.Context{}

// Reader is a wrapper around io.Reader, that allows to cancel read operation.
type Reader struct {
	ctx  context.Context
	data chan []byte

	sizeWant chan int

	err error
	r   io.Reader
}

func (c *Reader) begin() {
	for {
		select {
		case <-c.ctx.Done():
			for i, cn := range ctxes {
				if reflect.DeepEqual(c.ctx, cn) {
					ctxes = append(ctxes[:i], ctxes[i+1:]...)
					break
				}
			}
			close(c.data)
			close(c.sizeWant)
			return
		case sizeWant := <-c.sizeWant:
			buf := make([]byte, sizeWant)
			n, err := io.ReadFull(c.r, buf)
			if err != nil {
				c.err = err
				close(c.data)
				return
			}
			if n != sizeWant {
				panic("read " + strconv.Itoa(n) + ", want " + strconv.Itoa(sizeWant))
			}
			c.data <- buf
		}
	}
}

func (c *Reader) Read(p []byte) (int, error) {
	defer func() {
		if r := recover(); r != nil {
			c.err = io.ErrClosedPipe
		}
	}()

	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	case c.sizeWant <- len(p):
	}

	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	case d, ok := <-c.data:
		if !ok {
			return 0, c.err
		}
		copy(p, d)
		return len(d), nil
	}
}

func (c *Reader) ReadByte() (byte, error) {
	b := make([]byte, 1)

	n, err := c.Read(b)
	if err != nil {
		return 0x0, err
	}
	if n != 1 {
		panic(fmt.Errorf("read more than 1 byte, got %v", n))
	}

	return b[0], nil
}

func NewReader(ctx context.Context, r io.Reader) *Reader {
	ctxes = append(ctxes, ctx)

	c := &Reader{
		r:        r,
		ctx:      ctx,
		data:     make(chan []byte),
		sizeWant: make(chan int),
	}
	go c.begin()
	return c
}

func init() {
	http.HandleFunc("/ctx", func(w http.ResponseWriter, r *http.Request) {
		// for each ctx , print if it is done or not
		for _, ctx := range ctxes {
			fmt.Fprintf(w, "%v: %v\n", ctx, ctx.Err())
		}
	})
}