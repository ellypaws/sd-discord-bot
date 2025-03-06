package novelai

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"stable_diffusion_bot/api/novelai"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/queue"
	"sync"
	"time"
)

func New(token *string) queue.Queue[*NAIQueueItem] {
	if token == nil {
		return nil
	}
	return &NAIQueue{
		client:     novelai.NewNovelAIClient(*token),
		queue:      make(chan *NAIQueueItem, 24),
		cancelled:  make(map[string]bool),
		compositor: composite_renderer.Compositor(),
	}
}

type NAIQueue struct {
	client *novelai.Client

	botSession *discordgo.Session

	queue     chan *NAIQueueItem
	current   *NAIQueueItem
	cancelled map[string]bool
	mu        sync.Mutex

	compositor composite_renderer.Renderer

	stop chan os.Signal
}

func (q *NAIQueue) Start(botSession *discordgo.Session) {
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
				log.Printf("Waiting for current NovelAI to finish...")
				once = false
			}
		}
	}

	log.Printf("Polling stopped for NovelAI")
}

func (q *NAIQueue) Add(item *NAIQueueItem) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == cap(q.queue) {
		return -1, errors.New("queue is full")
	}

	item.pos = len(q.queue)
	q.queue <- item

	return item.pos, nil
}

func (q *NAIQueue) Remove(messageInteraction *discordgo.MessageInteractionMetadata) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Mark the item as cancelled
	q.cancelled[messageInteraction.ID] = true

	return nil
}

func (q *NAIQueue) Interrupt(i *discordgo.Interaction) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.current == nil {
		return errors.New("no generation to interrupt")
	}

	log.Printf("Interrupting generation #%s\n", q.current.DiscordInteraction.ID)
	if q.current.Interrupt == nil {
		q.current.Interrupt = make(chan *discordgo.Interaction)
	}
	q.current.Interrupt <- i
	close(q.current.Interrupt)

	return nil
}

func (q *NAIQueue) Stop() {
	if q.stop == nil {
		q.stop = make(chan os.Signal)
	}
	q.stop <- os.Interrupt
	close(q.stop)
}

func (q *NAIQueue) Commands() []*discordgo.ApplicationCommand { return q.commands() }

func (q *NAIQueue) Handlers() queue.CommandHandlers { return q.handlers() }

func (q *NAIQueue) Components() queue.Components { return q.components() }
