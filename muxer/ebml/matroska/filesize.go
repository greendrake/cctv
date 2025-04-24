package matroska

type writerFileSize struct {
	w    WriteSeekCloser
	size int
	seek bool
}

func (c *writerFileSize) FileSize() int {
	return c.size
}

func (c *writerFileSize) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)
	if !c.seek && err == nil {
		c.size += n
	}
	return
}

func (c *writerFileSize) Seek(offset int64, whence int) (int64, error) {
	c.seek = true
	return c.w.Seek(offset, whence)
}

func (c *writerFileSize) Close() error {
	return c.w.Close()
}
