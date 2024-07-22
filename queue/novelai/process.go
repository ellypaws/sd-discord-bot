package novelai

import (
	"fmt"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
)

func (q *NAIQueue) next() {
	for len(q.queue) > 0 {
		if q.current != nil {
			log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
			return
		}
		select {
		case q.current = <-q.queue:
			if q.current.DiscordInteraction == nil {
				log.Panicf("DiscordInteraction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", q.current)
				return
			}
			if i := q.current.DiscordInteraction; i != nil && q.cancelled[q.current.DiscordInteraction.ID] {
				// If the item is cancelled, skip it
				delete(q.cancelled, i.ID)
				q.done()
				return
			}
			switch q.current.Type {
			case ItemTypeImage:
				interaction, err := q.processCurrentItem()
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, interaction, fmt.Errorf("error processing current item: %w", err))
				}
			default:
				handlers.Errors[handlers.ErrorResponse](q.botSession, q.current.DiscordInteraction, fmt.Errorf("unknown item type: %v", q.current.Type))
				q.done()
			}
		}
	}
}

func (q *NAIQueue) done() {
	q.mu.Lock()
	q.current = nil
	q.mu.Unlock()
}
