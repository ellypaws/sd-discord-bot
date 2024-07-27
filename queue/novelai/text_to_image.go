package novelai

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/ellypaws/novelai-metadata/pkg/meta"
	"io"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
	"strings"
	"time"
)

func (q *NAIQueue) processCurrentItem() (*discordgo.Interaction, error) {
	defer q.done()
	item := q.current

	if item == nil {
		return nil, nil
	}

	if item.DiscordInteraction == nil {
		return nil, nil
	}

	request := item.Request
	if request == nil {
		return item.DiscordInteraction, errors.New("request is nil")
	}

	cost := request.CalculateCost(true)
	if cost >= 10 {
		return item.DiscordInteraction, fmt.Errorf("cost is %d", cost)
	}

	err := q.processImagineGrid(item)
	if err != nil {
		return item.DiscordInteraction, err
	}

	return item.DiscordInteraction, nil
}

func (q *NAIQueue) processImagineGrid(item *NAIQueueItem) error {
	request := item.Request

	embed, webhook, err := q.showInitialMessage(item)
	if err != nil {
		return err
	}

	generationDone := make(chan bool)
	go q.updateProgressBar(item, generationDone)

	switch item.Type {
	case ItemTypeImage, ItemTypeVibeTransfer, ItemTypeImg2Img:
		images, err := q.client.Inference(request)
		generationDone <- true
		if err != nil {
			return fmt.Errorf("error generating image: %w", err)
		}

		return q.showFinalMessage(item, images, embed, webhook)
	default:
		return fmt.Errorf("unknown item type: %s", item.Type)
	}
}

func (q *NAIQueue) showInitialMessage(queue *NAIQueueItem) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := queue.Request
	newContent := imagineMessageSimple(request, queue.user)

	embed := generationEmbedDetails(new(discordgo.MessageEmbed), queue, nil, queue.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message, err := handlers.EditInteractionResponse(q.botSession, queue.DiscordInteraction, webhook)
	if err != nil {
		return nil, nil, err
	}

	err = q.storeMessageInteraction(queue, message)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving message interaction: %v", err)
	}

	return embed, webhook, nil
}

func (q *NAIQueue) updateProgressBar(item *NAIQueueItem, generationDone <-chan bool) {
	start := time.Now()
	visual := spinner.Moon.Frames
	var frame int
	for {
		select {
		case item.DiscordInteraction = <-item.Interrupt:
			break
		case <-generationDone:
			fmt.Printf("\rFinished generating %s for %s in %s\n", item.DiscordInteraction.ID, item.user.Username, time.Since(start).Round(time.Second).String())
			return
		case <-time.After(1 * time.Second):
			frame = nextFrame(frame, len(visual))
			if frame >= len(visual) {
				frame = 0
			}
			message := imagineMessageSimple(item.Request, item.user)
			elapsed := time.Since(start).Round(time.Second).String()
			progress := fmt.Sprintf("\r%s\n\n%s Time elapsed: %s", message, visual[frame], elapsed)
			_, progressErr := q.botSession.InteractionResponseEdit(item.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &progress,
			})
			if progressErr != nil {
				log.Printf("Error editing interaction: %v", progressErr)
				return
			}
			fmt.Printf("\r%s Time elapsed: %s (%s)", visual[frame], elapsed, item.user.Username)
		case <-time.After(5 * time.Minute):
			log.Printf("Generation #%s has been running for 5 minutes, interrupting", item.DiscordInteraction.ID)
			break
		}
	}
}

func nextFrame(current, length int) int {
	return (current + 1) % length
}

func (q *NAIQueue) showFinalMessage(item *NAIQueueItem, response *entities.NovelAIResponse, embed *discordgo.MessageEmbed, webhook *discordgo.WebhookEdit) error {
	request := item.Request
	totalImages := int(request.Parameters.ImageCount)

	imageBuffers, thumbnailBuffers := retrieveImagesFromResponse(response, item)

	var user *discordgo.User
	if item.user != nil {
		user = item.user
	} else {
		user = &discordgo.User{ID: "unknown"}
	}
	mention := fmt.Sprintf("<@%v>", user.ID)
	// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
	embed = generationEmbedDetails(embed, item, getMetadata(response), item.Interrupt != nil)

	webhook = &discordgo.WebhookEdit{
		Content:    &mention,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.DeleteGeneration]},
	}

	if err := utils.EmbedImages(webhook, embed, imageBuffers[:min(len(imageBuffers), totalImages)], thumbnailBuffers, q.compositor); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	_, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, webhook)
	return err
}

func getMetadata(response *entities.NovelAIResponse) *meta.Metadata {
	if response == nil {
		return nil
	}

	for _, image := range response.Images {
		if image.Metadata != nil {
			return image.Metadata
		}
	}

	return nil
}

func retrieveImagesFromResponse(response *entities.NovelAIResponse, item *NAIQueueItem) (images []io.Reader, thumbnails []io.Reader) {
	images = make([]io.Reader, len(response.Images))

	for i, image := range response.Images {
		if image.Image == nil {
			log.Printf("error: image is nil at index %d\n", i)
			continue
		}

		reader, err := image.Reader()
		if err != nil {
			log.Printf("error encoding image: %s\n", err)
			continue
		}

		images[i] = reader
	}

	if item.Request.Parameters.VibeTransferImage != nil {
		reader, err := item.Request.Parameters.VibeTransferImage.Reader()
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, reader)
	}

	if item.Request.Parameters.Img2Img != nil {
		reader, err := item.Request.Parameters.Img2Img.Reader()
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, reader)
	}

	if len(images) > int(item.Request.Parameters.ImageCount) {
		thumbnails = append(thumbnails, images[item.Request.Parameters.ImageCount])
	}

	return images, thumbnails
}

func generationEmbedDetails(embed *discordgo.MessageEmbed, item *NAIQueueItem, metadata *meta.Metadata, interrupted bool) *discordgo.MessageEmbed {
	if item == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T", item)
		return embed
	}
	request := item.Request
	if request == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T or %T", request, item)
		return embed
	}
	if embed == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T, creating...", embed)
		embed = new(discordgo.MessageEmbed)
	}

	embed.Title = string(item.Type)
	if interrupted {
		embed.Title += " (Interrupted)"
	}
	embed.Type = discordgo.EmbedTypeImage
	embed.URL = "https://github.com/ellypaws/sd-discord-bot/"

	if item.user != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:         item.user.Username,
			IconURL:      item.user.AvatarURL(""),
			ProxyIconURL: "https://i.keiau.space/data/00144.png",
		}
	}

	timeSince := "unknown"
	if !item.Created.IsZero() {
		timeSince = time.Since(item.Created).Round(time.Second).String()
	}

	var user *discordgo.User
	if item.user != nil {
		user = item.user
	} else {
		user = &discordgo.User{ID: "unknown"}
	}
	embed.Description = fmt.Sprintf("<@%s> asked me to process `%v` images, `%v` steps in `%s`, cfg: `%0.1f`, seed: `%v`, sampler: `%s`",
		user.ID, request.Parameters.ImageCount, request.Parameters.Steps, timeSince,
		request.Parameters.Scale, request.Parameters.Seed, request.Parameters.Sampler)

	// store as "2015-12-31T12:00:00.000Z"
	embed.Timestamp = time.Now().Format(time.RFC3339)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text:    "https://github.com/ellypaws/sd-discord-bot/",
		IconURL: "https://i.keiau.space/data/00144.png",
	}

	if metadata != nil {
		generationTime := "`unknown`"
		if metadata.GenerationTime != nil {
			generationTime = fmt.Sprintf("`%ss`", (*metadata.GenerationTime)[:min(4, len(*metadata.GenerationTime))])
		}

		prompt := "unknown"
		if metadata.Description != "" {
			prompt = metadata.Description
		}
		if metadata.Comment != nil && metadata.Comment.Prompt != "" {
			prompt = metadata.Comment.Prompt
		}

		model := metadata.Source
		switch request.Model {
		case "":
			break
		case entities.ModelV3:
			model = "NAI Diffusion Anime V3"
		case entities.ModelFurryV3:
			model = "NAI Diffusion Furry V3"
		default:
			model = request.Model
		}
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Model",
				Value:  fmt.Sprintf("`%s`", model),
				Inline: false,
			},
			{
				Name:   "Generation Time",
				Value:  generationTime,
				Inline: true,
			},
			{
				Name:   "Seed",
				Value:  fmt.Sprintf("`%d`", metadata.Comment.Seed),
				Inline: true,
			},
			{
				Name:   "Steps",
				Value:  fmt.Sprintf("`%d`", metadata.Comment.Steps),
				Inline: true,
			},
			{
				Name:  "Prompt",
				Value: fmt.Sprintf("```\n%s\n```", prompt),
			},
		}
	} else {
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Model",
				Value:  fmt.Sprintf("`%s`", request.Model),
				Inline: false,
			},
			{
				Name:  "Prompt",
				Value: fmt.Sprintf("```\n%s\n```", request.Input),
			},
		}
	}

	return embed
}

func safeDereference(s *string) string {
	if s == nil {
		return "unknown"
	}
	return *s
}

func imagineMessageSimple(request *entities.NovelAIRequest, user *discordgo.User) string {
	var message strings.Builder

	seedString := fmt.Sprintf("%d", request.Parameters.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	if user == nil {
		user = &discordgo.User{ID: "unknown"}
	}
	message.WriteString(fmt.Sprintf("<@%s> asked me to imagine", user.ID))

	if message.Len() > 2000 {
		return message.String()[:2000]
	}

	return message.String()
}

// storeMessageInteraction stores the message interaction in the database to keep track of the message ID to recreate the message
func (q *NAIQueue) storeMessageInteraction(queue *NAIQueueItem, message *discordgo.Message) (err error) {
	//request := queue.Request
	//
	//if queue.DiscordInteraction == nil {
	//	return fmt.Errorf("queue.DiscordInteraction is nil")
	//}
	//
	//if message == nil {
	//	message, err = q.botSession.InteractionResponse(queue.DiscordInteraction)
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	//// store message ID in c.DiscordInteraction.Message
	//queue.DiscordInteraction.Message = message
	//
	//request.InteractionID = queue.DiscordInteraction.ID
	//request.MessageID = queue.DiscordInteraction.Message.ID
	//request.MemberID = queue.DiscordInteraction.Member.User.ID
	//request.SortOrder = 0
	//request.Processed = true
	return nil
}
