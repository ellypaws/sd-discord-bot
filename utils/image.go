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
	"sync"
)

// Image is an io.Reader that asynchronously downloads an image from a URL.
// The data returned by the Read method is the raw bytes, but MarshalJSON encodes in base64.StdEncoding
// The zero value of Image is not usable, use AsyncImage instead.
type Image struct {
	ch     chan []byte
	err    error
	buffer bytes.Buffer
	id     int
	closed bool
}

var asyncPool = sync.Pool{New: newAsync}

// AsyncImage returns an *Image that asynchronously downloads the image from the given URL as the object is created.
// The returned data is in base64 format.
func AsyncImage(url string) *Image {
	result := asyncPool.Get().(*Image)
	result.Reset()

	go func() {
		defer close(result.ch)
		data, err := GetDataFromUrl(url)
		if err != nil {
			result.err = err
			return
		}

		result.ch <- data
	}()

	return result
}

func (r *Image) Read(b []byte) (int, error) {
	r.flush()

	if r.err != nil {
		return 0, r.err
	}

	var i int
	i, r.err = r.buffer.Read(b)
	if r.err == io.EOF {
		defer r.close()
	}

	return i, r.err
}

func (r *Image) MarshalJSON() ([]byte, error) {
	r.flush()
	out := bytes.NewBuffer(make([]byte, 0, r.buffer.Len()+2))
	encoder := base64.NewEncoder(base64.StdEncoding, out)
	defer encoder.Close()

	out.WriteByte('"')
	_, err := r.buffer.WriteTo(encoder)
	if err != nil {
		return nil, err
	}
	out.WriteByte('"')
	return out.Bytes(), nil
}

// flush writes the data from the channel to the buffer, waiting until the data is ready.
// multiple calls to flush will simultaneously unlock once the channel is closed
func (r *Image) flush() {
	bin, ok := <-r.ch
	if ok {
		r.buffer.Write(bin)
	}
}

func (r *Image) Bytes() []byte {
	r.flush()
	return r.buffer.Bytes()
}

func (r *Image) String() string {
	r.flush()
	return r.buffer.String()
}

func (r *Image) Buffer() *bytes.Buffer {
	r.flush()
	return &r.buffer
}

func (r *Image) Reset() {
	r.ch = make(chan []byte)
	r.closed = false
	r.err = nil
	r.buffer.Reset()
}

var asyncID int

func newAsync() any {
	async := Image{id: asyncID}
	asyncID++
	return &async
}

func (r *Image) close() {
	if r.closed {
		return
	}
	r.err = io.EOF
	r.buffer.Reset()
	asyncPool.Put(r)
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

// DownloadImageAsBase64 downloads an image from the given URL and returns it as a base64 string.
// Deprecated: Use AsyncImage instead.
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
