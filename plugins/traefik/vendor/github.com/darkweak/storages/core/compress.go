package core

import (
	"bytes"
	"sync"

	"github.com/pierrec/lz4/v4"
)

// maxPooledBufferSize bounds the scratch buffers kept in compressBufferPool,
// so a single huge response does not pin its buffer in memory forever.
const maxPooledBufferSize = 4 << 20

// compressWriterPool pools lz4 writers configured with the smallest block
// size. The lz4 default block size is 4 MB, which makes every compression,
// and every decompression of the frame, use 4 MB blocks even for tiny
// payloads: a concurrent read holds its own 4 MB buffer until the body is
// fully read. Readers pick the block size up from the frame header, so the
// frames stay readable by any lz4 reader, including the ones written with the
// default block size. The block size lives in the frame descriptor, which
// Reset keeps.
var compressWriterPool = sync.Pool{New: func() any {
	writer := lz4.NewWriter(nil)
	_ = writer.Apply(lz4.BlockSizeOption(lz4.Block64Kb))

	return writer
}}

var compressBufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// CompressValue returns value compressed as a single lz4 frame, the format
// readResponse expects. The returned slice is exactly sized and owned by the
// caller, so it can be retained, e.g. by in-memory storers, without pinning
// any scratch memory.
func CompressValue(value []byte) ([]byte, error) {
	buffer, _ := compressBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()

	writer, _ := compressWriterPool.Get().(*lz4.Writer)
	writer.Reset(buffer)

	_, err := writer.Write(value)
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}

	// Close flushed the frame, the writer is safe to reuse whatever happened.
	writer.Reset(nil)
	compressWriterPool.Put(writer)

	var compressed []byte
	if err == nil {
		compressed = bytes.Clone(buffer.Bytes())
	}

	if buffer.Cap() <= maxPooledBufferSize {
		compressBufferPool.Put(buffer)
	}

	return compressed, err
}
