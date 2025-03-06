package utils

import (
	"errors"
	"reflect"

	"github.com/bwmarrin/discordgo"
)

func GetUsername(entities ...any) string {
	if user := GetUser(entities...); user != nil {
		return user.Username
	}
	return "unknown"
}

func GetUser(entities ...any) *discordgo.User {
	for _, entity := range entities {
		v := reflect.ValueOf(entity)
		if v.Kind() == reflect.Pointer && v.IsNil() {
			continue
		}
		switch e := entity.(type) {
		case *discordgo.User:
			return e
		case *discordgo.Member:
			return GetUser(e.User)
		case *discordgo.Message:
			return GetUser(e.Author, e.Member)
		case *discordgo.MessageCreate:
			return GetUser(e.Message)
		case *discordgo.MessageUpdate:
			return GetUser(e.Message, e.BeforeUpdate)
		case *discordgo.MessageDelete:
			return GetUser(e.Message, e.BeforeDelete)
		case *discordgo.Interaction:
			return GetUser(e.Member, e.User)
		case *discordgo.InteractionCreate:
			return GetUser(e.Interaction)
		case *discordgo.MessageInteraction:
			return GetUser(e.User, e.Member)
		case *discordgo.MessageInteractionMetadata:
			return GetUser(e.User)
		default:
			continue
		}
	}
	return nil
}

func GetReference(s *discordgo.Session, r *discordgo.MessageReference) (*discordgo.Message, error) {
	if s == nil {
		return nil, errors.New("*discordgo.Session is nil")
	}
	if r == nil {
		return nil, errors.New("*discordgo.MessageReference is nil")
	}
	retrieve, err := s.State.Message(r.ChannelID, r.MessageID)
	if err != nil {
		if errors.Is(err, discordgo.ErrStateNotFound) {
			retrieve, err = s.ChannelMessage(r.ChannelID, r.MessageID)
			if err != nil {
				return nil, err
			}
			err = s.State.MessageAdd(retrieve)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return retrieve, nil
}
