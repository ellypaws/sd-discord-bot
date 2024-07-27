package novelai

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
)

type CloseAfterRead struct {
	Reader io.ReadCloser
}

func (c *CloseAfterRead) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	if err == io.EOF {
		c.Reader.Close()
	}
	return n, err
}

func Unzip(body io.ReadCloser) ([]io.Reader, error) {
	bin, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	byteReader := bytes.NewReader(bin)
	zipReader, err := zip.NewReader(byteReader, byteReader.Size())
	if err != nil {
		return nil, err
	}

	if len(zipReader.File) == 0 {
		return nil, errors.New("zip file is empty")
	}

	images := make([]io.Reader, len(zipReader.File))
	for i, file := range zipReader.File {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		images[i] = &CloseAfterRead{Reader: reader}
		//data, err := meta.ExtractMetadata(img)
		//if err != nil {
		//	continue
		//}
		//
		//valid, err := data.IsNovelAI()
		//if err != nil {
		//	return nil, err
		//}
		//
		//if !valid {
		//	continue
		//}
		//
		//images[i].Metadata = data
	}
	return images, nil
}

func GZIP(data []byte) (*bytes.Buffer, error) {
	compressed := new(bytes.Buffer)
	zipper := gzip.NewWriter(compressed)

	_, err := zipper.Write(data)
	if err != nil {
		return nil, err
	}

	err = zipper.Close()
	if err != nil {
		return nil, err
	}

	return compressed, nil
}
