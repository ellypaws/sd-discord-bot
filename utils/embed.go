package utils

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"stable_diffusion_bot/composite_renderer"
	"time"
)

// EmbedImages modifies the provided webhook to include the provided embed and images.
// If there are more than four images, they will be tiled into a single image.
// images and thumbnails are expected to be in bytes and not base64 encoded.
func EmbedImages(webhook *discordgo.WebhookEdit, embed *discordgo.MessageEmbed, images, thumbnails []io.Reader, compositor composite_renderer.Renderer) error {
	if webhook == nil {
		return errors.New("imageEmbedFromBuffers called with nil webhook")
	}
	now := time.Now().UTC()
	nowFormatted := now.Format("2006-01-02_15-04-05")
	if embed == nil {
		embed = &discordgo.MessageEmbed{
			Type:      discordgo.EmbedTypeImage,
			URL:       "https://github.com/ellypaws/sd-discord-bot/",
			Timestamp: now.Format(time.RFC3339),
		}
	}

	var files []*discordgo.File
	var embeds []*discordgo.MessageEmbed

	embeds = append(embeds, embed)

	// Process thumbnails
	for i := len(thumbnails) - 1; i >= 0; i-- {
		if thumbnails[i] == nil {
			thumbnails = append(thumbnails[:i], thumbnails[i+1:]...)
		}
	}

	if len(thumbnails) > 0 {
		thumbnailTile, err := composite_renderer.Compositor().TileImages(thumbnails)
		if err != nil {
			return fmt.Errorf("error tiling thumbnails: %w", err)
		}
		if thumbnailTile != nil {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: "attachment://thumbnail.png",
			}
			files = append(files, &discordgo.File{
				Name:   "thumbnail.png",
				Reader: thumbnailTile,
			})
		}
	}

	for i := len(images) - 1; i >= 0; i-- {
		if images[i] == nil {
			images = append(images[:i], images[i+1:]...)
		}
	}

	// Process primary images
	if len(images) > 4 {
		// Tile images if more than four
		if compositor == nil {
			return errors.New("compositor is required for tiling more than four images")
		}
		primaryTile, err := compositor.TileImages(images)
		if err != nil {
			return fmt.Errorf("error tiling primary images: %w", err)
		}
		imgName := fmt.Sprintf("%v.png", nowFormatted)
		files = append(files, &discordgo.File{
			Name:   imgName,
			Reader: primaryTile,
		})
		embed.Image = &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%s", imgName),
		}
	} else {
		// Create separate embeds for four or fewer images
		for i, imgBuf := range images {
			if imgBuf == nil {
				continue
			}

			imgName := fmt.Sprintf("%v-%d.png", nowFormatted, i)
			files = append(files, &discordgo.File{
				Name:   imgName,
				Reader: imgBuf,
			})

			newEmbed := &discordgo.MessageEmbed{
				Type: discordgo.EmbedTypeImage,
				URL:  embed.URL, // Using the same URL as the original embed
				Image: &discordgo.MessageEmbedImage{
					URL: fmt.Sprintf("attachment://%s", imgName),
				},
			}
			if i == 0 {
				embed.Image = newEmbed.Image
			} else {
				embeds = append(embeds, newEmbed)
			}
		}
	}

	webhook.Embeds = &embeds
	webhook.Files = files
	return nil
}
