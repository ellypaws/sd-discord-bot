package novelai

import (
	"archive/zip"
	"bytes"
	"errors"
	"github.com/ellypaws/novelai-metadata/pkg/meta"
	"image"
	"io"
	"stable_diffusion_bot/entities"
)

func Unzip(body io.ReadCloser) ([]entities.Image, error) {
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

	images := make([]entities.Image, len(zipReader.File))
	for i, file := range zipReader.File {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		img, _, err := image.Decode(reader)
		if err != nil {
			return nil, err
		}

		images[i] = entities.Image{Image: &img}

		bin := new(bytes.Buffer)
		err = images[i].Encode(bin)
		if err != nil {
			continue
		}
		data, err := meta.ExtractMetadata(bin)
		if err != nil {
			continue
		}

		valid, err := meta.IsNovelAI(*data)
		if err != nil {
			return nil, err
		}

		if !valid {
			continue
		}

		images[i].Metadata = data
	}
	return images, nil
}
