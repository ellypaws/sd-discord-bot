package utils

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
)

const (
	port = ":5724"
	url  = "http://localhost" + port + "/file.txt"
)

var file []byte = []byte("success")

func init() {
	http.HandleFunc("/file.txt", func(w http.ResponseWriter, r *http.Request) {
		i, err := io.Copy(w, bytes.NewReader(file))
		if err != nil {
			panic(err)
		}
		log.Printf("Sent %d bytes", i)
	})

	go func() {
		err := http.ListenAndServe(port, nil)
		if err != nil {
			panic(err)
		}
	}()
}

func TestImage(t *testing.T) {
	image := new(Image)
	// testing zero value
	_, err := image.Read([]byte{})
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}

	image.Download(url)

	b := make([]byte, image.Len())
	i, err := image.Read(b)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	t.Logf("Read %d bytes", i)
}

func TestAsyncImage(t *testing.T) {
	image := AsyncImage(url)

	data := make([]byte, 1024)
	n, err := image.Read(data)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Fatalf("no data read")
	}
}

func TestRunMultiple(t *testing.T) {
	// run TestDownload
	for range 10 {
		t.Run("TestDownload", func(t *testing.T) {
			TestDownload(t)
		})
	}
}

func TestDownload(t *testing.T) {
	image := &Image{}
	image.Download(url)

	var bin bytes.Buffer
	n, err := io.Copy(&bin, image)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Fatalf("no data read")
	}
	if n != int64(len(file)) {
		t.Fatalf("mismatched data length: %d vs %d", n, len(file))
	}
	if bin.Len() != len(file) {
		t.Fatalf("mismatched data length: %d vs %d", bin.Len(), len(file))
	}

	err = os.WriteFile("download.txt", bin.Bytes(), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(bin.Bytes(), file) {
		t.Fatalf("mismatched data")
	}
}

func TestMarshalJSON(t *testing.T) {
	image := AsyncImage(url)

	jsonData, err := image.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.HasPrefix(jsonData, []byte(`"c3VjY2Vz"`)) {
		t.Fatalf("unexpected JSON output: %s", string(jsonData[:min(len(jsonData), 10)]))
	}
}

func TestMultipleReads(t *testing.T) {
	image := AsyncImage(url)

	data1 := make([]byte, len(file)/2)
	n1, err := image.Read(data1)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	data2 := make([]byte, len(file)/2)
	n2, err := image.Read(data2)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	if n1 != n2 {
		t.Fatalf("mismatched data lengths: %d vs %d", n1, n2)
	}
}

func TestLen(t *testing.T) {
	image := AsyncImage(url)

	length := image.Len()
	if length <= 0 {
		t.Fatalf("invalid length: %d", length)
	}
}

func TestString(t *testing.T) {
	image := AsyncImage(url)

	str := image.String()
	if len(str) == 0 {
		t.Fatalf("unexpected empty string")
	}
}

func TestBuffer(t *testing.T) {
	image := AsyncImage(url)

	buffer := image.Buffer()
	if buffer.Len() <= 0 {
		t.Fatalf("invalid buffer length")
	}
}

func TestConcurrentAccess(t *testing.T) {
	image := AsyncImage(url)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := make([]byte, 1024)
			_, err := image.Read(data)
			if err != nil && err != io.EOF {
				t.Fatalf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestGetDataBody(t *testing.T) {
	body, err := GetDataBody(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	data := make([]byte, 1024)
	n, err := body.Read(data)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Fatalf("no data read")
	}
}

func TestImageReset(t *testing.T) {
	image := &Image{}
	image.reset()

	if !image.open {
		t.Fatalf("expected image to be open")
	}
	if image.err != nil {
		t.Fatalf("expected no error, got %v", image.err)
	}
	if image.buffer.Len() != 0 {
		t.Fatalf("expected empty buffer, got %d bytes", image.buffer.Len())
	}
}

func TestImageClose(t *testing.T) {
	image := &Image{}
	image.reset()
	image.close()

	if image.open {
		t.Fatalf("expected image to be closed")
	}
	if image.err != io.EOF {
		t.Fatalf("expected EOF error, got %v", image.err)
	}
	if image.buffer.Len() != 0 {
		t.Fatalf("expected empty buffer, got %d bytes", image.buffer.Len())
	}
}

func TestUseTwice(t *testing.T) {
	image := AsyncImage(url)

	for range 2 {
		bin, err := image.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bin) < 3 {
			t.Fatalf("unexpected empty data")
		}
		t.Logf("%s", bin)
	}
}
