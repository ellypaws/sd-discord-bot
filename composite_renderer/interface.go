package composite_renderer

import (
	"io"
)

type Renderer interface {
	TileImages(imageBufs []io.Reader) (io.Reader, error)
}

// New returns a new Renderer. Set yonsai to true if you have 4 images to render, false if you have n images to render.
func New(yonsai bool) Renderer {
	if yonsai {
		return &rendererImpl{}
	} else {
		return &tilerImpl{}
	}
}

func Compositor() Renderer {
	return &compositor{}
}
