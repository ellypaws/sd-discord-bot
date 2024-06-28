package entities

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ellypaws/inkbunny-sd/llm"
	"time"
)

type QueueItem struct {
	Type ItemType

	*ImageGenerationRequest

	LLMRequest *llm.Request
	LLMCreated time.Time

	AspectRatio        string
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction

	ADetailerString string // use AppendSegModelByString
	Attachments     map[string]*MessageAttachment

	Img2ImgItem
	ControlnetItem

	Raw *TextToImageRaw // raw JSON input

	Interrupt chan *discordgo.Interaction
}

type Img2ImgItem struct {
	*MessageAttachment
	DenoisingStrength float64
}

type ControlnetItem struct {
	*MessageAttachment
	ControlMode  ControlMode
	ResizeMode   ResizeMode
	Type         string
	Preprocessor string // also called the module in entities.ControlNetParameters
	Model        string
	Enabled      bool
}

type ItemType int
