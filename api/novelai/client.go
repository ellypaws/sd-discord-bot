package novelai

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"stable_diffusion_bot/entities"
)

type Client struct {
	token token
	host  url.URL
}

func NewNovelAIClient(key string) *Client {
	return &Client{
		token: token(key),
		host: url.URL{
			Scheme: "https",
			Host:   "image.novelai.net",
			Path:   "/ai/generate-image",
		},
	}
}

func (c *Client) Inference(request *entities.NovelAIRequest) (*entities.NovelAIResponse, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	bin, err := request.Marshal()
	if err != nil {
		return nil, err
	}

	response, err := c.POST(bin)
	if err != nil {
		return nil, err
	}

	return &entities.NovelAIResponse{Images: response}, nil
}

func (c *Client) POST(bin []byte) ([]entities.Image, error) {
	request, err := http.NewRequest(http.MethodPost, c.host.String(), bytes.NewReader(bin))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	c.token.setAuth(&request.Header)

	client := new(http.Client)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		errorString := "(unknown error)"

		body, _ := io.ReadAll(response.Body)
		if len(body) > 0 {
			errorString = fmt.Sprintf("\n```json\n%v\n```", string(body))
		}

		return nil, fmt.Errorf("unexpected status code: %d %s", response.StatusCode, errorString)
	}

	contentType := response.Header.Get("Content-Type")
	switch contentType {
	case "application/zip":
		return Unzip(response.Body)
	case "binary/octet-stream":
		return Unzip(response.Body)
	default:
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}
}

type token string

type Setter interface {
	Set(string, string)
}

func (t token) String() string {
	return fmt.Sprintf("Bearer %s", string(t))
}

func (t token) setAuth(req Setter) {
	req.Set("Authorization", t.String())
}
