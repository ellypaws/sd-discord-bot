package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

type (
	Command       string
	CommandOption string
)

var (
	// Command names
	helloCommand           Command = "hello"
	imagineCommand         Command = "imagine"
	imagineSettingsCommand Command = "imagine_settings"
)

const (
	// Command options
	promptOption       CommandOption = "prompt"
	negativeOption     CommandOption = "negative_prompt"
	samplerOption      CommandOption = "sampler_name"
	aspectRatio        CommandOption = "aspect_ratio"
	loraOption         CommandOption = "lora"
	checkpointOption   CommandOption = "checkpoint"
	vaeOption          CommandOption = "vae"
	hypernetworkOption CommandOption = "hypernetwork"
	embeddingOption    CommandOption = "embedding"
	hiresFixOption     CommandOption = "use_hires_fix"
	hiresFixSize       CommandOption = "hires_fix_size"
	restoreFacesOption CommandOption = "restore_faces"
	adModelOption      CommandOption = "ad_model"
	cfgScaleOption     CommandOption = "cfg_scale"

	extraLoras = 6
)

var commands = map[Command]*discordgo.ApplicationCommand{
	helloCommand: {
		Name: string(helloCommand),
		// All commands and options must have a description
		// Commands/options without description will fail the registration
		// of the command.
		Description: "Say hello to the bot",
		Type:        discordgo.ChatApplicationCommand,
	},
	imagineCommand: {
		Name:        string(imagineCommand),
		Description: "Ask the bot to imagine something",
		Options:     imagineOptions(),
	},
	imagineSettingsCommand: {
		Name:        string(imagineSettingsCommand),
		Description: "Change the default settings for the imagine command",
	},
}

func imagineOptions() (options []*discordgo.ApplicationCommandOption) {
	options = []*discordgo.ApplicationCommandOption{
		commandOptions[promptOption],
		commandOptions[negativeOption],
		commandOptions[checkpointOption],
		commandOptions[aspectRatio],
		commandOptions[loraOption],
		commandOptions[samplerOption],
		commandOptions[hiresFixOption],
		commandOptions[hiresFixSize],
		commandOptions[cfgScaleOption],
		commandOptions[restoreFacesOption],
		commandOptions[adModelOption],
		commandOptions[vaeOption],
		commandOptions[hypernetworkOption],
		commandOptions[embeddingOption],
	}

	for i := 0; i < extraLoras; i++ {
		loraOption := *commandOptions[loraOption]
		loraOption.Name += fmt.Sprintf("%d", i+2)
		options = append(options, &loraOption)
	}
	return
}

var commandOptions = map[CommandOption]*discordgo.ApplicationCommandOption{
	promptOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(promptOption),
		Description: "The text prompt to imagine",
		Required:    true,
	},
	negativeOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(negativeOption),
		Description: "Negative prompt",
		Required:    false,
	},
	checkpointOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         string(checkpointOption),
		Description:  "The checkpoint to change to when generating. Sets for the next person.",
		Required:     false,
		Autocomplete: true,
	},
	vaeOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         string(vaeOption),
		Description:  "The vae to use",
		Required:     false,
		Autocomplete: true,
	},
	hypernetworkOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         string(hypernetworkOption),
		Description:  "The hypernetwork to use",
		Required:     false,
		Autocomplete: true,
	},
	embeddingOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         string(embeddingOption),
		Description:  "The embedding to use",
		Required:     false,
		Autocomplete: true,
	},
	aspectRatio: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(aspectRatio),
		Description: "The aspect ratio to use. Default is 1:1 (note: you can specify your own aspect ratio)",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1:1",
				Value: "1:1",
			},
			{
				Name:  "2:3",
				Value: "2:3",
			},
			{
				Name:  "3:2",
				Value: "3:2",
			},
			{
				Name:  "3:4",
				Value: "3:4",
			},
			{
				Name:  "4:3",
				Value: "4:3",
			},
			{
				Name:  "16:9",
				Value: "16:9",
			},
			{
				Name:  "9:16",
				Value: "9:16",
			},
		},
	},
	loraOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         string(loraOption),
		Description:  "The lora(s) to apply",
		Required:     false,
		Autocomplete: true,
	},
	samplerOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(samplerOption),
		Description: "sampler",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Euler a",
				Value: "Euler a",
			},
			{
				Name:  "DDIM",
				Value: "DDIM",
			},
			{
				Name:  "UniPC",
				Value: "UniPC",
			},
			{
				Name:  "Euler",
				Value: "Euler",
			},
			{
				Name:  "DPM2 a Karras",
				Value: "DPM2 a Karras",
			},
			{
				Name:  "DPM++ 2S a Karras",
				Value: "DPM++ 2S a Karras",
			},
			{
				Name:  "DPM++ 2M Karras",
				Value: "DPM++ 2M Karras",
			},
			{
				Name:  "DPM++ 3M SDE Karras",
				Value: "DPM++ 3M SDE Karras",
			},
		},
	},
	hiresFixOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(hiresFixOption),
		Description: "use hires.fix or not. default=No for better performance",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Yes",
				Value: "true",
			},
			{
				Name:  "No",
				Value: "false",
			},
		},
	},
	hiresFixSize: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(hiresFixSize),
		Description: "upscale multiplier for hires.fix. default=2",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1.5",
				Value: "1.5",
			},
			{
				Name:  "2",
				Value: "2",
			},
		},
	},
	cfgScaleOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        string(cfgScaleOption),
		Description: "upscale multiplier for cfg. default=7",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "7",
				Value: "7",
			},
		},
	},
	restoreFacesOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(restoreFacesOption),
		Description: "Use Codeformer to restore faces",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Yes",
				Value: "true",
			},
			{
				Name:  "No",
				Value: "false",
			},
		},
	},
	adModelOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        string(adModelOption),
		Description: "The model to use for adetailer",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Face",
				Value: "face_yolov8n.pt",
			},
			{
				Name:  "Body",
				Value: "person_yolov8n-seg.pt",
			},
			{
				Name:  "Both",
				Value: "person_yolov8n-seg.pt,face_yolov8n.pt",
			},
		},
	},
}

const (
	maskedUser    = "user"
	maskedChannel = "channel"
	maskedForum   = "threads"
	maskedRole    = "role"
)

var maskedOptions = map[string]*discordgo.ApplicationCommandOption{
	maskedUser: {
		Type:        discordgo.ApplicationCommandOptionUser,
		Name:        maskedUser,
		Description: "Choose a user",
		Required:    false,
	},
	maskedChannel: {
		Type:        discordgo.ApplicationCommandOptionChannel,
		Name:        maskedChannel,
		Description: "Choose a channel to close",
		// Channel type mask
		ChannelTypes: []discordgo.ChannelType{
			discordgo.ChannelTypeGuildText,
			discordgo.ChannelTypeGuildVoice,
		},
		Required: false,
	},
	maskedForum: {
		Type:        discordgo.ApplicationCommandOptionChannel,
		Name:        maskedForum,
		Description: "Choose a thread to mark as solved",
		ChannelTypes: []discordgo.ChannelType{
			discordgo.ChannelTypeGuildForum,
			discordgo.ChannelTypeGuildNewsThread,
			discordgo.ChannelTypeGuildPublicThread,
			discordgo.ChannelTypeGuildPrivateThread,
		},
	},
	maskedRole: {
		Type:        discordgo.ApplicationCommandOptionRole,
		Name:        maskedRole,
		Description: "Choose a role to add",
		Required:    false,
	},
}

// addImagineCommand is now inside the commands map as imagineCommand: commands[imagineCommand]
// It also uses imagineOptions() to build the necessary commandOptions
// Deprecated: use commands[imagineCommand]
func (b *botImpl) addImagineCommand(name string, command *discordgo.ApplicationCommand) (error, *discordgo.ApplicationCommand) {
	log.Printf("Adding command '%s'...", name)

	commands[imagineCommand].Options = imagineOptions()

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, commands[imagineCommand])
	if err != nil {
		log.Printf("Error creating '%s' command: %v", name, err)

		return err, nil
	}

	return nil, cmd
}

// Deprecated: use commandHandlers[imagineCommand]
func (b *botImpl) addImagineSettingsCommand(command string) (error, *discordgo.ApplicationCommand) {
	log.Printf("Adding command '%s'...", command)

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        string(b.imagineSettingsCommandString()),
		Description: "Change the default settings for the imagine command",
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineSettingsCommandString(), err)

		return err, nil
	}

	//b.registeredCommands[command] = cmd

	return nil, cmd
}
