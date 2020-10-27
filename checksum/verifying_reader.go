package checksum

import (
	"errors"
	"fmt"
	"hash"
	"io"
)

type VerifyingReader struct {
	reader   io.Reader
	hash     hash.Hash
	checkSum string
}

func NewVerifyingReader(reader io.Reader, hash hash.Hash, checkSum string) *VerifyingReader {
	return &VerifyingReader{reader: io.TeeReader(reader, hash), hash: hash, checkSum: checkSum}
}

func (r *VerifyingReader) Read(p []byte) (n int, err error) {
	bytesRead, err := r.reader.Read(p)
	if errors.Is(err, io.EOF) {
		if checkSumErr := r.verifyChecksum(fmt.Sprintf("%x", r.hash.Sum(nil))); checkSumErr != nil {
			return bytesRead, checkSumErr
		}

		return bytesRead, io.EOF
	}

	if err != nil {
		return bytesRead, fmt.Errorf("reader-error: %w", err)
	}

	return bytesRead, nil
}

func (r *VerifyingReader) verifyChecksum(actualCheckSum string) error {
	if r.checkSum == actualCheckSum {
		return nil
	}

	return errors.New("checksum verification failure")
}
