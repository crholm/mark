package tsar

import (
	"errors"
	"io"
)

func newByteReadSeeker(data []byte) io.ReadSeeker{
	return &byteReadSeeker{
		data:   data,
		offset: 0,
	}
}

type byteReadSeeker struct {
	data   []byte
	offset int64
}

func (mf *byteReadSeeker) Read(p []byte) (n int, err error) {

	if mf.offset == int64(len(mf.data)) {
		return 0, io.EOF
	}

	n = copy(p, mf.data[mf.offset:])
	mf.offset += int64(n)

	return
}

func (mf *byteReadSeeker) Seek(offset int64, whence int) (ret int64, err error) {
	var relativeTo int64
	switch whence {
	case 0:
		relativeTo = 0
	case 1:
		relativeTo = mf.offset
	case 2:
		relativeTo = int64(len(mf.data))
	}
	ret = relativeTo + offset
	if ret < 0 || ret > int64(len(mf.data)) {
		return -1, errors.New("New offset would fall outside of the byteReadSeeker")
	}
	mf.offset = ret
	return
}
