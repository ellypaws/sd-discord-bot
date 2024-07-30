package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
)

func AsyncImage(url string) *Image {
	result := Image{ch: make(chan []byte)}

	go func() {
		defer close(result.ch)
		data, err := DownloadImageAsBase64(url)
		if err != nil {
			result.err = err
			return
		}

		result.ch <- []byte(data)
	}()

	return &result
}

type Image struct {
	ch     chan []byte
	err    error
	buffer bytes.Buffer
}

func (r *Image) Read(b []byte) (int, error) {
	bin, ok := <-r.ch
	if ok {
		r.buffer.Write(bin)
	}

	if r.err != nil {
		return 0, r.err
	}

	return r.buffer.Read(b)
}

func GetDataFromUrl(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func DownloadImageAsBase64(url string) (string, error) {
	imageData, err := GetDataFromUrl(url)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(imageData), nil
}

func Base64ToByteReader(base64Str string) (*bytes.Reader, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func GetBase64ImageSize(base64Str string) (int, int, error) {
	// Cut "data:image/*;base64," prefix, if present.
	before, after, found := strings.Cut(base64Str, ";base64,")

	trimmed := after
	if !found {
		trimmed = before
	}

	reader, err := Base64ToByteReader(trimmed)
	if err != nil {
		return 0, 0, err
	}

	img, _, err := image.Decode(reader)
	if err != nil {
		return 0, 0, err
	}

	boundSize := img.Bounds().Size()

	return boundSize.X, boundSize.Y, nil
}
