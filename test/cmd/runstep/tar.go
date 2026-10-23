package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
)

// untarSingle returns the contents of the first regular file in a tar stream,
// which is what `docker cp <container>:<file> -` produces.
func untarSingle(data []byte) (string, error) {
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return "", errors.New("no file in archive")
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		content, err := io.ReadAll(tr)
		return string(content), err
	}
}
