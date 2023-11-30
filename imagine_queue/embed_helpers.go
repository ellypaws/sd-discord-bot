package imagine_queue

import (
	"bytes"
	"fmt"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/entities"
	"time"
)

func imageEmbedFromAttachment(webhook *discordgo.WebhookEdit, embed *discordgo.MessageEmbed, image *entities.MessageAttachment, thumbnail *bytes.Reader) (err error) {
	if embed == nil {
		embed = &discordgo.MessageEmbed{
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	var files []*discordgo.File

	if thumbnail != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "attachment://thumbnail.png",
		}
		files = append(files, &discordgo.File{
			Name:   "thumbnail.png",
			Reader: thumbnail,
		})
	}

	var reader *bytes.Reader
	if image != nil {
		if image.Image == nil && image.URL != "" {
			image.Image = new(string)
			*image.Image, err = utils.GetImageBase64(image.URL)
			if err != nil {
				log.Printf("Error getting image base64: %v", err)
				return
			}
		}
		embed.Type = discordgo.EmbedTypeImage
		embed.Image = &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%v", image.Filename),
		}
	}

	reader, err = utils.GetImageReaderByBase64(safeDereference(image.Image))
	if err != nil {
		log.Printf("Error getting image reader by base64: %v", err)
		return
	}

	files = append(files, &discordgo.File{
		Name:   image.Filename,
		Reader: reader,
	})

	embeds := []*discordgo.MessageEmbed{embed}

	webhook.Embeds = &embeds
	webhook.Files = files
	return
}

func imageEmbedFromReader(webhook *discordgo.WebhookEdit, embed *discordgo.MessageEmbed, reader *bytes.Reader, thumbnail *bytes.Reader) (err error) {
	if embed == nil {
		embed = &discordgo.MessageEmbed{
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	var files []*discordgo.File

	if thumbnail != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "attachment://thumbnail.png",
		}
		files = append(files, &discordgo.File{
			Name:   "thumbnail.png",
			Reader: thumbnail,
		})
	}

	embed.Type = discordgo.EmbedTypeImage
	embed.Image = &discordgo.MessageEmbedImage{
		URL: fmt.Sprintf("attachment://%v", "image.png"),
	}

	files = append(files, &discordgo.File{
		Name:   "image.png",
		Reader: reader,
	})

	embeds := []*discordgo.MessageEmbed{embed}

	webhook.Embeds = &embeds
	webhook.Files = files
	return
}

func generationEmbedDetails(embed *discordgo.MessageEmbed, newGeneration *entities.ImageGenerationRequest, c *QueueItem) *discordgo.MessageEmbed {
	if newGeneration == nil || c == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T or %T", newGeneration, c)
		return nil
	}
	var title string
	switch {
	case c.Type == ItemTypeImg2Img || (c.Img2ImgItem.MessageAttachment != nil && c.Img2ImgItem.Image != nil):
		title = "Image to Image"
	case c.ControlnetItem.MessageAttachment != nil && c.ControlnetItem.Image != nil:
		title = "Controlnet"
	case c.Type == ItemTypeVariation:
		title = "Variation"
	case c.Type == ItemTypeReroll:
		title = "Reroll"
	case c.Type == ItemTypeUpscale:
		title = "Upscale"
	default:
		title = "Finished generation"
	}
	if embed == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T", embed)
		embed = &discordgo.MessageEmbed{}
	}
	embed.Type = discordgo.EmbedTypeImage
	embed.Title = title
	embed.URL = "https://github.com/ellypaws/sd-discord-bot/"
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name:         c.DiscordInteraction.Member.User.Username,
		IconURL:      c.DiscordInteraction.Member.User.AvatarURL(""),
		ProxyIconURL: "https://i.keiau.space/data/00144.png",
	}
	embed.Description = fmt.Sprintf("<@%s> asked me to process %v steps in %v, cfg: `%0.1f`, seed: `%v`, sampler: `%s`",
		c.DiscordInteraction.Member.User.ID, newGeneration.Steps, time.Since(newGeneration.CreatedAt),
		newGeneration.CFGScale, newGeneration.Seed, newGeneration.SamplerName)
	// store as "2015-12-31T12:00:00.000Z"
	embed.Timestamp = time.Now().Format(time.RFC3339)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text:    "https://github.com/ellypaws/sd-discord-bot/",
		IconURL: "https://i.keiau.space/data/00144.png",
	}
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Checkpoint",
			Value:  fmt.Sprintf("`%v`", safeDereference(newGeneration.Checkpoint)),
			Inline: false,
		},
		{
			Name:   "VAE",
			Value:  fmt.Sprintf("`%v`", safeDereference(newGeneration.VAE)),
			Inline: true,
		},
		{
			Name:   "Hypernetwork",
			Value:  fmt.Sprintf("`%v`", safeDereference(newGeneration.Hypernetwork)),
			Inline: true,
		},
		{
			Name:  "Prompt",
			Value: fmt.Sprintf("```\n%s\n```", newGeneration.Prompt),
		},
	}
	return embed
}

func rerollVariationComponents(amount int) *[]discordgo.MessageComponent {
	var actionsRow []discordgo.ActionsRow

	var firstRow []discordgo.MessageComponent

	// First Row: "imagine_variation" buttons and "Re-roll" button
	for i := 1; i <= amount; i++ {
		firstRow = append(firstRow, discordgo.Button{
			Label:    fmt.Sprintf("%d", i),
			Style:    discordgo.SecondaryButton,
			Disabled: false,
			CustomID: fmt.Sprintf("imagine_variation_%d", i),
			Emoji: discordgo.ComponentEmoji{
				Name: "♻️",
			},
		})
	}

	firstRow = append(firstRow, discordgo.Button{
		Label:    "Re-roll",
		Style:    discordgo.PrimaryButton,
		Disabled: false,
		CustomID: "imagine_reroll",
		Emoji: discordgo.ComponentEmoji{
			Name: "🎲",
		},
	})

	actionsRow = append(actionsRow, discordgo.ActionsRow{
		Components: firstRow,
	})

	var secondRow []discordgo.MessageComponent

	// Second Row: "imagine_upscale" buttons and "Delete" button
	for i := 1; i <= amount; i++ {
		secondRow = append(secondRow, discordgo.Button{
			Label:    fmt.Sprintf("%d", i),
			Style:    discordgo.SecondaryButton,
			Disabled: false,
			CustomID: fmt.Sprintf("imagine_upscale_%d", i),
			Emoji: discordgo.ComponentEmoji{
				Name: "⬆️",
			},
		})
	}

	// "Delete" button
	secondRow = append(secondRow, discordgo.Button{
		Label:    "Delete",
		Style:    discordgo.DangerButton,
		Disabled: false,
		CustomID: "imagine_delete",
		Emoji: discordgo.ComponentEmoji{
			Name: "🗑️",
		},
	})

	actionsRow = append(actionsRow, discordgo.ActionsRow{
		Components: secondRow,
	})

	// Create the ActionsRows
	var rows []discordgo.MessageComponent
	for _, row := range actionsRow {
		rows = append(rows, row)
	}

	return &rows
}
