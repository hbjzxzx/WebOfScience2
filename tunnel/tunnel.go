package tunnel

import (
	"io"
)

type Tunnel interface {
	io.Writer
	io.Reader
}
