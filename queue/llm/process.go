package llm

import (
	"fmt"
	"log"

	"stable_diffusion_bot/discord_bot/handlers"
)

func (q *LLMQueue) next() error {
	for len(q.queue) > 0 {
		if q.current != nil {
			log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
			return fmt.Errorf("currentImagine is not nil")
		}
		select {
		case q.current = <-q.queue:
			if q.current.DiscordInteraction == nil {
				log.Panicf("DiscordInteraction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", q.current)
			}

			if i := q.current.DiscordInteraction; i != nil && q.cancelled[q.current.DiscordInteraction.ID] {
				// If the item is cancelled, skip it
				delete(q.cancelled, i.ID)
				q.done()
				return nil
			}
			switch q.current.Type {
			case ItemTypeInstruct:
				err := q.processLLM()
				if err != nil {
					return fmt.Errorf("error processing current item: %w", err)
				}
			default:
				q.done()
				return handlers.ErrorEdit(q.botSession, q.current.DiscordInteraction, fmt.Errorf("unknown item type: %s", q.current.Type))
			}
		}
	}
	return nil
}

func (q *LLMQueue) done() {
	q.mu.Lock()
	q.current = nil
	q.mu.Unlock()
}
