package stable_diffusion

import (
	"fmt"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func generationEmbedDetails(embed *discordgo.MessageEmbed, queue *SDQueueItem, interrupted bool) *discordgo.MessageEmbed {
	if queue == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T", queue)
		return embed
	}
	request := queue.ImageGenerationRequest
	if request == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T or %T", request, queue)
		return embed
	}
	if embed == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T, creating...", embed)
		embed = &discordgo.MessageEmbed{}
	}
	switch {
	case queue.Enabled && queue.Type == ItemTypeImg2Img:
		embed.Title = "Image to Image (Controlnet)"
	case queue.Enabled || (queue.ControlnetItem.MessageAttachment != nil && queue.ControlnetItem.Image != nil):
		embed.Title = "Text to Image (Controlnet)"
	case queue.Type == ItemTypeImg2Img || (queue.Img2ImgItem.MessageAttachment != nil && queue.Img2ImgItem.Image != nil):
		embed.Title = "Image to Image"
	case queue.Type == ItemTypeVariation:
		embed.Title = "Variation"
	case queue.Type == ItemTypeReroll:
		embed.Title = "Reroll"
	case queue.Type == ItemTypeUpscale:
		embed.Title = "Upscale"
	case queue.Type == ItemTypeRaw:
		embed.Title = "JSON to Image"
	default:
		embed.Title = "Text to Image"
	}
	if interrupted {
		embed.Title += " (Interrupted)"
	}
	embed.Type = discordgo.EmbedTypeImage
	embed.URL = "https://github.com/ellypaws/sd-discord-bot/"
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name:         queue.DiscordInteraction.Member.User.Username,
		IconURL:      queue.DiscordInteraction.Member.User.AvatarURL(""),
		ProxyIconURL: "https://i.keiau.space/data/00144.png",
	}

	var timeSince string
	if request.CreatedAt.IsZero() {
		timeSince = "unknown"
	} else {
		timeSince = time.Since(request.CreatedAt).Round(time.Second).String()
	}

	embed.Description = fmt.Sprintf("<@%s> asked me to process `%v` images, `%v` steps in %v, cfg: `%0.1f`, seed: `%v`, sampler: `%s`",
		queue.DiscordInteraction.Member.User.ID, request.NIter*request.BatchSize, request.Steps, timeSince,
		request.CFGScale, request.Seed, request.SamplerName)

	var scripts []string

	if queue.Type != ItemTypeRaw {
		if request.Scripts.ADetailer != nil {
			scripts = append(scripts, "ADetailer")
		}
		if request.Scripts.ControlNet != nil {
			scripts = append(scripts, "ControlNet")
		}
		if request.Scripts.CFGRescale != nil {
			scripts = append(scripts, "CFGRescale")
		}
	} else {
		for script := range queue.Raw.RawScripts {
			scripts = append(scripts, script)
		}
	}

	if len(scripts) > 0 {
		embed.Description += fmt.Sprintf("\n**Scripts**: [`%v`]", strings.Join(scripts, ", "))
	}

	if request.OverrideSettings.CLIPStopAtLastLayers > 1 {
		embed.Description += fmt.Sprintf("\n**CLIPSkip**: `%v`", request.OverrideSettings.CLIPStopAtLastLayers)
	}

	// store as "2015-12-31T12:00:00.000Z"
	embed.Timestamp = time.Now().Format(time.RFC3339)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text:    "https://github.com/ellypaws/sd-discord-bot/",
		IconURL: "https://i.keiau.space/data/00144.png",
	}
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Checkpoint",
			Value:  fmt.Sprintf("`%v`", safeDereference(request.Checkpoint)),
			Inline: false,
		},
		{
			Name:   "VAE",
			Value:  fmt.Sprintf("`%v`", safeDereference(request.VAE)),
			Inline: true,
		},
		{
			Name:   "Hypernetwork",
			Value:  fmt.Sprintf("`%v`", safeDereference(request.Hypernetwork)),
			Inline: true,
		},
		{
			Name:  "Prompt",
			Value: fmt.Sprintf("```\n%s\n```", request.Prompt),
		},
	}
	if queue.Raw != nil && queue.Raw.Debug {
		// remove prompt, last item from embed.Fields
		embed.Fields = embed.Fields[:len(embed.Fields)-1]
	}
	return embed
}

// rerollVariationComponents returns a buttons with discordgo.MessageComponent with a specified image count.
// A maximum of 4 buttons will be returned (due to Discord's limit) plus one "Re-roll" or "Delete" button.
// If disable is true, the Variation and Upscale buttons will be disabled.
func rerollVariationComponents(amount int, disable bool) *[]discordgo.MessageComponent {
	amount = min(amount, 4)

	var actionsRow []discordgo.ActionsRow

	var firstRow []discordgo.MessageComponent

	// First Row: "imagine_variation" buttons and "Re-roll" button
	for i := 1; i <= amount; i++ {
		firstRow = append(firstRow, discordgo.Button{
			Label:    fmt.Sprintf("%d", i),
			Style:    discordgo.SecondaryButton,
			Disabled: disable,
			CustomID: fmt.Sprintf("%v_%d", VariantButton, i),
			Emoji: &discordgo.ComponentEmoji{
				Name: "♻️",
			},
		})
	}

	firstRow = append(firstRow, discordgo.Button{
		Label:    "Re-roll",
		Style:    discordgo.PrimaryButton,
		Disabled: disable,
		CustomID: RerollButton,
		Emoji: &discordgo.ComponentEmoji{
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
			Disabled: disable,
			CustomID: fmt.Sprintf("%v_%d", UpscaleButton, i),
			Emoji: &discordgo.ComponentEmoji{
				Name: "⬆️",
			},
		})
	}

	// "Delete" button
	secondRow = append(secondRow, discordgo.Button{
		Label:    "Delete",
		Style:    discordgo.DangerButton,
		Disabled: false,
		CustomID: handlers.DeleteGeneration,
		Emoji: &discordgo.ComponentEmoji{
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
