package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/databases/sqlite"
	"stable_diffusion_bot/discord_bot"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/queue/llm"
	"stable_diffusion_bot/queue/novelai"
	"stable_diffusion_bot/queue/stable_diffusion"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"strings"

	_ "stable_diffusion_bot/pprof"

	openai "github.com/ellypaws/inkbunny-sd/llm"
	"github.com/joho/godotenv"
)

// Bot parameters
var (
	guildID            = flag.String("guild", "", "Guild ID. If not passed - bot registers commands globally")
	botToken           = flag.String("token", "", "Bot access token")
	apiHost            = flag.String("host", "", "Host for the Automatic1111 API")
	imagineCommand     = flag.String("imagine", "imagine", "Imagine command name. Default is \"imagine\"")
	removeCommandsFlag = flag.Bool("remove", false, "Delete all commands when bot exits")

	llmHost      = flag.String("llm", "", "LLM model to use")
	novelAIToken = flag.String("novelai", "", "NovelAI API token")
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	log.Println(".env file loaded successfully")

	if botToken == nil || *botToken == "" {
		tokenEnv := os.Getenv("BOT_TOKEN")
		if tokenEnv == "YOUR_BOT_TOKEN_HERE" {
			log.Fatalf("Invalid bot token from .env file: %v\n"+
				"Did you edit the .env or run the program with -token ?", tokenEnv)
		}
		if tokenEnv != "" {
			botToken = &tokenEnv
		}
	}

	if apiHost == nil || *apiHost == "" {
		hostEnv := os.Getenv("API_HOST")
		if hostEnv != "" {
			apiHost = &hostEnv
		}
	}

	if apiHost != nil && *apiHost != "" {
		sanitized := strings.TrimSuffix(*apiHost, "/")
		apiHost = &sanitized
	}

	if guildID == nil || *guildID == "" {
		guildEnv := os.Getenv("GUILD_ID")
		if guildEnv != "" {
			guildID = &guildEnv
		}
	}

	if imagineCommand == nil || *imagineCommand == "" {
		imagineEnv := os.Getenv("IMAGINE_COMMAND")
		if imagineEnv != "" {
			*imagineCommand = imagineEnv
		}
	}

	if llmHost == nil || *llmHost == "" {
		llmHostEnv := os.Getenv("LLM_HOST")
		if llmHostEnv != "" {
			llmHost = &llmHostEnv
		}
	}

	if novelAIToken == nil || *novelAIToken == "" {
		novelAITokenEnv := os.Getenv("NOVELAI_TOKEN")
		if novelAITokenEnv != "" {
			novelAIToken = &novelAITokenEnv
		}
	}

	if removeCommandsFlag == nil || !*removeCommandsFlag {
		removeCommandsEnv := os.Getenv("REMOVE_COMMANDS")
		if removeCommandsEnv != "" {
			removeCommandsFlag = new(bool)
			*removeCommandsFlag = removeCommandsEnv == "true"
		}
	}
}

func main() {
	flag.Parse()

	//if guildID == nil || *guildID == "" {
	//	log.Fatalf("Guild ID flag is required")
	//}

	if botToken == nil || *botToken == "" {
		log.Fatalf("Bot token flag is required")
	}

	if apiHost == nil || *apiHost == "" {
		log.Fatalf("API host flag is required")
	}

	alive := handlers.CheckAPIAlive(*apiHost)
	if !alive {
		log.Printf("API (%v) is not running! Continuing anyway...", *apiHost)
	}

	if imagineCommand == nil || *imagineCommand == "" {
		log.Fatalf("Imagine command flag is required")
	}

	var removeCommands bool

	if removeCommandsFlag != nil && *removeCommandsFlag {
		removeCommands = *removeCommandsFlag
	}

	stableDiffusionAPI, err := stable_diffusion_api.New(stable_diffusion_api.Config{
		Host: *apiHost,
	})
	if err != nil {
		log.Fatalf("Failed to create Stable Diffusion API: %v", err)
	}

	errors := stableDiffusionAPI.PopulateCache()
	if errors != nil {
		log.Printf("Failed to populate cache: %v", err)
	}

	ctx := context.Background()

	sqliteDB, err := sqlite.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create sqlite database: %v", err)
	}

	generationRepo, err := image_generations.NewRepository(&image_generations.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create image generation repository: %v", err)
	}

	defaultSettingsRepo, err := default_settings.NewRepository(&default_settings.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create default settings repository: %v", err)
	}

	imagineQueue, err := stable_diffusion.New(stable_diffusion.Config{
		StableDiffusionAPI:  stableDiffusionAPI,
		ImageGenerationRepo: generationRepo,
		DefaultSettingsRepo: defaultSettingsRepo,
	})
	if err != nil {
		log.Fatalf("Failed to create imagine queue: %v", err)
	}

	var llmConfig *openai.Config
	if llmHost != nil && *llmHost != "" {
		endpoint, err := url.Parse(*llmHost)
		if err != nil {
			log.Fatalf("Failed to parse LLM host: %v", err)
		}
		llmConfig = &openai.Config{
			Host:     *llmHost,
			APIKey:   "", // TODO: Add API key
			Endpoint: *endpoint,
		}
		log.Printf("LLM host set to %s", llmConfig.Endpoint.String())
	} else {
		log.Printf("LLM host is not set, LLM commands will be disabled")
	}

	bot, err := discord_bot.New(&discord_bot.Config{
		BotToken:           *botToken,
		GuildID:            *guildID,
		ImagineQueue:       imagineQueue,
		NovelAIQueue:       novelai.New(novelAIToken),
		LLMQueue:           llm.New(llmConfig),
		RemoveCommands:     removeCommands,
		StableDiffusionApi: stableDiffusionAPI,
	})
	if err != nil {
		log.Fatalf("Error creating Discord bot: %v", err)
	}

	if err := bot.Start(); err != nil {
		panic(err)
	}

	log.Println("Gracefully shutting down.")
}
