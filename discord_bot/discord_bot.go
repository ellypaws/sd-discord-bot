package discord_bot

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"os/signal"
	"slices"
	"sync"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/queue/llm"
	"stable_diffusion_bot/queue/novelai"
	"stable_diffusion_bot/queue/stable_diffusion"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	botSession *discordgo.Session

	registeredCommands map[handlers.Command]*discordgo.ApplicationCommand
	config             *Config

	queues []queue.HandlerStartStopper

	handlers   queue.CommandHandlers
	components queue.Components
}

type Config struct {
	BotToken           string
	GuildID            string
	ImagineQueue       queue.Queue[*stable_diffusion.SDQueueItem]
	NovelAIQueue       queue.Queue[*novelai.NAIQueueItem]
	LLMQueue           queue.Queue[*llm.LLMItem]
	RemoveCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
}

func New(cfg *Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	handlers.Token = &cfg.BotToken

	if cfg.GuildID == "" {
		//return nil, errors.New("missing guild ID")
		log.Printf("Guild ID not provided, commands will be registered globally")
	}

	if cfg.ImagineQueue == nil {
		return nil, errors.New("missing imagine queue")
	}

	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	queues := []queue.HandlerStartStopper{
		cfg.ImagineQueue,
		cfg.NovelAIQueue,
		cfg.LLMQueue,
	}
	queues = slices.DeleteFunc(queues, func(q queue.HandlerStartStopper) bool { return q == nil })

	bot := &botImpl{
		botSession:         botSession,
		registeredCommands: make(map[handlers.Command]*discordgo.ApplicationCommand),
		config:             cfg,
		queues:             queues,
		handlers:           make(queue.CommandHandlers),
		components:         handlers.ComponentHandlers,
	}

	return bot, nil
}

func (b *botImpl) registerHandlers() {
	for _, q := range b.queues {
		handlers := q.Handlers()
		for interactionType, commandHandlers := range handlers {
			if _, ok := b.handlers[interactionType]; !ok {
				maps.Copy(b.handlers, handlers)
			} else {
				maps.Copy(b.handlers[interactionType], commandHandlers)
			}
		}

		maps.Copy(b.components, q.Components())
	}

	b.botSession.AddHandler(func(session *discordgo.Session, i *discordgo.InteractionCreate) {
		var handler queue.Handler
		var ok bool
		if i.Type == discordgo.InteractionMessageComponent {
			log.Printf("Component with customID `%v` was pressed, attempting to respond\n", i.MessageComponentData().CustomID)
			handler, ok = b.components[i.MessageComponentData().CustomID]
		} else {
			handles, exist := b.handlers[i.Type]
			if !exist {
				log.Printf("Unknown interaction type: %v", i.Type)
				return
			}

			handler, ok = handles[i.ApplicationCommandData().Name]
		}

		if !ok || handler == nil {
			var interactionType = "unknown"
			var interactionName = "unknown"
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				interactionType = "command"
				interactionName = i.ApplicationCommandData().Name
			case discordgo.InteractionMessageComponent:
				interactionType = "component"
				interactionName = i.MessageComponentData().CustomID
			case discordgo.InteractionApplicationCommandAutocomplete:
				interactionType = "autocomplete"

				data := i.ApplicationCommandData()
				for _, opt := range data.Options {
					if !opt.Focused {
						continue
					}
					interactionName = fmt.Sprintf("command: /%s option: %s", data.Name, opt.Name)
					break
				}
			case discordgo.InteractionModalSubmit:
				interactionType = "modal"
				interactionName = i.ModalSubmitData().CustomID
			}
			log.Printf("WARNING: Cannot find handler for interaction [%v] '%v'", interactionType, interactionName)
			return
		}

		err := handler(session, i)

		var username = "unknown"
		if i.Member != nil {
			username = i.Member.User.Username
		}
		if i.User != nil {
			username = i.User.Username
		}

		if err != nil {
			if errors.Is(err, handlers.ResponseError) {
				log.Printf("Error responding to interaction for %s: %v", username, err)
				return
			}
			err := handlers.ErrorEdit(session, i.Interaction, err)
			if err != nil {
				log.Printf("Error showing error message to user %s: %v", username, err)
			}
		}
	})
}

func (b *botImpl) registerCommands() error {
	b.registeredCommands = make(map[handlers.Command]*discordgo.ApplicationCommand)

	for _, q := range b.queues {
		if q == nil {
			continue
		}

		for _, command := range q.Commands() {
			cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.config.GuildID, command)
			if err != nil {
				return fmt.Errorf("cannot create '%s' command: %w", command.Name, err)
			}

			b.registeredCommands[command.Name] = cmd
			log.Printf("Registered %v command as: /%v", command.Name, cmd.Name)
		}
	}

	return nil
}

func (b *botImpl) Start() error {
	b.botSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err := b.botSession.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	err = b.registerCommands()
	if err != nil {
		return fmt.Errorf("error registering commands: %w", err)
	}

	b.registerHandlers()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	queues := []queue.StartStop{
		b.config.ImagineQueue,
		b.config.NovelAIQueue,
		b.config.LLMQueue,
	}

	queues = slices.DeleteFunc(queues, IsNil)
	for _, q := range queues {
		go q.Start(b.botSession)
	}

	if len(queues) == 0 {
		log.Println("No queues to start, exiting...")
		stop <- os.Interrupt
	} else {
		log.Println("Press Ctrl+C to exit")
	}

	<-stop
	var wg sync.WaitGroup
	for _, q := range queues {
		wg.Add(1)
		go func(q queue.StartStop) {
			q.Stop()
			wg.Done()
		}(q)
	}
	wg.Wait()

	err = b.teardown()
	if err != nil {
		return fmt.Errorf("error tearing down bot: %w", err)
	}

	return nil
}

func IsNil(q queue.StartStop) bool {
	return q == nil
}

func (b *botImpl) teardown() error {
	// Delete all commands added by the bot
	if b.config.RemoveCommands {
		log.Printf("Removing all commands added by bot...")

		for key, v := range b.registeredCommands {
			log.Printf("Removing command [key:%v], '%v'...", key, v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.config.GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	return b.botSession.Close()
}
