package llm

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/ellypaws/inkbunny-sd/llm"
	"log"
	"os"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/queue"
	"sync"
	"time"
)

func New(host *llm.Config) queue.Queue[*LLMItem] {
	if host == nil {
		return nil
	}
	return &LLMQueue{
		host:       host,
		queue:      make(chan *LLMItem, 24),
		cancelled:  make(map[string]bool),
		compositor: composite_renderer.Compositor(),
	}
}

type LLMQueue struct {
	host *llm.Config

	botSession *discordgo.Session

	queue     chan *LLMItem
	current   *LLMItem
	cancelled map[string]bool
	mu        sync.Mutex

	compositor composite_renderer.Renderer

	stop chan os.Signal
}

func (q *LLMQueue) Start(botSession *discordgo.Session) {
	q.botSession = botSession

	var once bool

Polling:
	for {
		select {
		case <-q.stop:
			break Polling
		case <-time.After(1 * time.Second):
			if q.current == nil {
				if err := q.next(); err != nil {
					log.Printf("Error processing next item: %v", err)
				}
				once = true
			} else if once {
				log.Printf("Waiting for current LLM to finish...")
				once = false
			}
		}
	}

	log.Printf("Polling stopped for LLM")
}

func (q *LLMQueue) Add(item *LLMItem) (int, error) {
	if len(q.queue) == cap(q.queue) {
		return -1, errors.New("queue is full")
	}

	q.queue <- item

	return len(q.queue), nil
}

func (q *LLMQueue) Remove(messageInteraction *discordgo.MessageInteraction) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Mark the item as cancelled
	q.cancelled[messageInteraction.ID] = true

	return nil
}

func (q *LLMQueue) Interrupt(i *discordgo.Interaction) error {
	if q.current == nil {
		return errors.New("no generation to interrupt")
	}

	log.Printf("Interrupting generation #%s\n", q.current.DiscordInteraction.ID)
	if q.current.Interrupt == nil {
		q.current.Interrupt = make(chan *discordgo.Interaction)
	}
	q.current.Interrupt <- i

	return nil
}

func (q *LLMQueue) Stop() {
	if q.stop == nil {
		q.stop = make(chan os.Signal)
	}
	q.stop <- os.Interrupt
	close(q.stop)
}

func (q *LLMQueue) Commands() []*discordgo.ApplicationCommand {
	return q.commands()
}

func (q *LLMQueue) Handlers() queue.CommandHandlers { return q.handlers() }

func (q *LLMQueue) Components() queue.Components { return q.components() }
