package novelai

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/utils"
)

const NovelAICommand = "novelai"

const (
	promptOption   = "prompt"
	negativeOption = "negative_prompt"
	cfgScaleOption = "cfg_scale"
	stepOption     = "step"
	seedOption     = "seed"

	novelaiModelOption    = "model"
	novelaiSizeOption     = "size"
	novelaiSamplerOption  = "sampler"
	novelaiUCPresetOption = "uc_preset"
	novelaiQualityOption  = "quality"
	novelaiScheduleOption = "schedule"
	novelaiSMEAOption     = "smea"
	novelaiSMEADynOption  = "smea_dyn"

	novelaiVibeTransfer = "vibe_transfer"
	novelaiInformation  = "information_extracted"
	novelaiReference    = "reference_strength"
	novelaiImg2ImgStr   = "img2img_strength"

	img2imgOption   = "img2img"
	denoisingOption = "denoising"
)

func (q *NAIQueue) handlers() queue.CommandHandlers {
	return queue.CommandHandlers{
		discordgo.InteractionApplicationCommand: {
			NovelAICommand: q.processNovelAICommand,
		},
	}
}

func (q *NAIQueue) processNovelAICommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	optionMap := utils.GetOpts(i.ApplicationCommandData())
	option, ok := optionMap[promptOption]
	if !ok {
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a prompt.")
	}

	item := q.NewItem(i.Interaction, WithPrompt(option.StringValue()))
	item.Type = ItemTypeImage

	if option, ok = optionMap[negativeOption]; ok {
		item.Request.Parameters.NegativePrompt = option.StringValue()
	}

	if option, ok = optionMap[novelaiModelOption]; ok {
		item.Request.Model = option.StringValue()
	}

	if option, ok = optionMap[novelaiSamplerOption]; ok {
		item.Request.Parameters.Sampler = option.StringValue()
	}

	if option, ok = optionMap[seedOption]; ok {
		item.Request.Parameters.Seed = option.IntValue()
	}

	if option, ok = optionMap[novelaiSizeOption]; ok {
		preset := entities.GetDimensions(option.StringValue())
		item.Request.Parameters.ResolutionPreset = &preset
	}

	if option, ok = optionMap[novelaiSMEAOption]; ok {
		item.Request.Parameters.Smea = option.BoolValue()
	}

	if option, ok = optionMap[novelaiSMEADynOption]; ok {
		item.Request.Parameters.SmeaDyn = option.BoolValue()
	}

	if option, ok = optionMap[novelaiUCPresetOption]; ok {
		value := option.IntValue()
		item.Request.Parameters.UcPreset = &value
	}

	if option, ok = optionMap[novelaiQualityOption]; ok {
		item.Request.Parameters.QualityToggle = option.BoolValue()
	}

	if option, ok = optionMap[cfgScaleOption]; ok {
		item.Request.Parameters.Scale = option.FloatValue()
	}

	if option, ok = optionMap[novelaiScheduleOption]; ok {
		item.Request.Parameters.NoiseSchedule = option.StringValue()
	}

	attachments, err := utils.GetAttachments(i)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error getting attachments.", err)
	}

	if option, ok := optionMap[novelaiVibeTransfer]; ok {
		attachment, ok := attachments[option.Value.(string)]
		if !ok {
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide an image to img2img.")
		}
		if item.Request.Model == entities.ModelV4Preview {
			return handlers.ErrorEdit(s, i.Interaction, "Vibe transfer is not yet supported for V4 models.")
		}

		item.Type = ItemTypeVibeTransfer
		item.Request.Parameters.VibeTransferImage = attachment.Image

		if option, ok := optionMap[novelaiInformation]; ok {
			item.Request.Parameters.ReferenceInformationExtracted = option.FloatValue()
		}

		if option, ok := optionMap[novelaiReference]; ok {
			item.Request.Parameters.ReferenceStrength = option.FloatValue()
		}
	}

	if option, ok := optionMap[img2imgOption]; ok {
		image, ok := attachments[option.Value.(string)]
		if !ok {
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide an image to img2img.")
		}

		item.Type = ItemTypeImg2Img
		item.Request.Action = entities.ActionImg2Img
		item.Request.Parameters.Img2Img = image.Image

		if option, ok := optionMap[novelaiImg2ImgStr]; ok {
			item.Request.Parameters.Strength = option.FloatValue()
		}
	}

	_, err = q.Add(item)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error adding imagine to queue.", err)
	}

	message, err := handlers.EditInteractionResponse(s, i.Interaction,
		q.positionString(item),
		handlers.Components[handlers.Cancel],
	)
	if err != nil {
		return err
	}

	if item.DiscordInteraction != nil && item.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting message ID for interaction %v", item.DiscordInteraction.ID)
		item.DiscordInteraction.Message = message
	}

	return nil
}

func (q *NAIQueue) positionString(item *NAIQueueItem) string {
	snowflake := utils.GetUser(item.DiscordInteraction).ID
	if item.pos <= 0 {
		return fmt.Sprintf(
			"I'm dreaming something up for you. You are next in line.\n<@%s> asked me to imagine \n```\n%s\n```",
			snowflake,
			item.Request.Input,
		)
	} else {
		return fmt.Sprintf(
			"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \n```\n%s\n```",
			item.pos,
			snowflake,
			item.Request.Input,
		)
	}
}
