package utils

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func GetOpts(data discordgo.ApplicationCommandInteractionData) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	options := data.Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

// keyValue matches --key value, --key=value, or --key "value with spaces"
var keyValue = regexp.MustCompile(`\B(?:--|—)+(\w+)(?:[ =]([\w./\\:]+|"[^"]+"))?`)

func ExtractKeyValuePairsFromPrompt(prompt string) (parameters map[string]string, sanitized string) {
	parameters = make(map[string]string)
	sanitized = keyValue.ReplaceAllString(prompt, "")
	sanitized = strings.TrimSpace(sanitized)
	for _, match := range keyValue.FindAllStringSubmatch(prompt, -1) {
		parameters[match[1]] = match[2]
	}
	return
}

// InterfaceConvertAuto supplies field values from the interaction data or parameters map.
// If FieldType and ValueType are the same, then we attempt to assert FieldType to value.Value
// Otherwise, we return the interface conversion to the caller to do manual type conversion
//
// Example:
//
//	if int64Val, ok := interfaceConvertAuto[int, int64](&queue.Steps, stepOption, optionMap, parameters); ok {
//		queue.Steps = int(*int64Val)
//	}
//
// (*discordgo.ApplicationCommandInteractionDataOption).IntValue() actually uses float64 for the interface conversion, so use float64 for integers, numbers, etc.
// and then convert to the desired type.
// Only string and float64 are supported for V as that's what the discordgo API returns.
// If the field is nil, then we don't assign the value to the field.
// Instead, we return *V and bool to indicate whether the conversion was successful.
// This is useful for when we want to convert to a type that is not the same as the field type.
func InterfaceConvertAuto[F any, V string | float64](field *F, option string, optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption, parameters map[string]string) (*V, bool) {
	if value, ok := optionMap[option]; ok {
		vToField, ok := value.Value.(F)
		if ok && field != nil {
			*field = vToField
		}
		valueType, ok := value.Value.(V)
		return &valueType, ok
	}
	if value, ok := parameters[option]; ok {
		if field != nil {
			_, err := fmt.Sscanf(value, "%v", field)
			if err != nil {
				return nil, false
			}
		}
		var out V
		_, err := fmt.Sscanf(value, "%v", &out)
		if err != nil {
			return nil, false
		}
		return &out, true
	}
	return nil, false
}

type AttachmentImage struct {
	Attachment *discordgo.MessageAttachment
	Image      *Image
}

func GetAttachments(i *discordgo.InteractionCreate) (map[string]AttachmentImage, error) {
	if i.ApplicationCommandData().Resolved == nil {
		return nil, nil
	}

	resolved := i.ApplicationCommandData().Resolved.Attachments
	if resolved == nil {
		return nil, nil
	}

	attachments := make(map[string]AttachmentImage, len(resolved))
	for snowflake, attachment := range resolved {
		log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
		if !strings.HasPrefix(attachment.ContentType, "image") {
			log.Printf("Attachment[%v] is not an image, removing from queue.", snowflake)
			continue
		}

		attachments[snowflake] = AttachmentImage{attachment, AsyncImage(attachment.URL)}
	}

	return attachments, nil
}
