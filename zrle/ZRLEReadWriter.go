package zrle

import (
	"bytes"
	"compress/zlib"
	"io"
	"log"
)

type ZlibStream struct {
	zlibReader io.ReadCloser
	buffer     *bytes.Buffer
}

func (z *ZlibStream) Read(p []byte) (n int, e error) {

	if z.zlibReader == nil {
		log.Println("  no zlib reader found, creating one...")
		z.zlibReader, _ = zlib.NewReader(z.buffer)
	}
	n, e = z.zlibReader.Read(p)
	//log.Printf("      zlib read: %v bytes, %v remaining", n, z.buffer.Len())

	return
}

func (z *ZlibStream) Write(p []byte) (n int, e error) {
	log.Printf("  zlib write - %v bytes", len(p))
	if z.buffer == nil {
		log.Println("  no zlib buffer found, creating one...")
		z.buffer = bytes.NewBuffer([]byte{})
	}
	n, e = z.buffer.Write(p)
	return
}
