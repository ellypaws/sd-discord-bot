package novelai

import (
	"fmt"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"time"
)

func (q *NAIQueue) next() error {
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
			case ItemTypeImage, ItemTypeVibeTransfer, ItemTypeImg2Img:
				interaction, err := q.processCurrentItem()
				if err != nil {
					if interaction == nil {
						return err
					}
					return handlers.ErrorEdit(q.botSession, interaction, fmt.Errorf("error processing current item: %w", err))
				}
			default:
				q.done()
				return handlers.ErrorEdit(q.botSession, q.current.DiscordInteraction, fmt.Errorf("unknown item type: %s", q.current.Type))
			}
		}
	}
	return nil
}

func (q *NAIQueue) done() {
	q.mu.Lock()
	q.current = nil
	q.updateWaiting()
	q.mu.Unlock()
}

// updateWaiting updates all queued items with their new position
func (q *NAIQueue) updateWaiting() {
	items := len(q.queue)
	finished := make(chan *NAIQueueItem, items)
	defer close(finished)

	for range items {
		go func(item *NAIQueueItem) {
			item.pos--
			var queueString string
			if item.pos == 0 {
				queueString = fmt.Sprintf(
					"I'm dreaming something up for you. You are next in line.\n<@%s> asked me to imagine \n```\n%s\n```",
					item.DiscordInteraction.Member.User.ID,
					item.Request.Input,
				)
			} else {
				queueString = fmt.Sprintf(
					"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \n```\n%s\n```",
					item.pos,
					item.DiscordInteraction.Member.User.ID,
					item.Request.Input,
				)
			}
			_, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, queueString, handlers.Components[handlers.Cancel])
			if err != nil {
				log.Printf("Error updating queue position for item %v: %v", item.DiscordInteraction.ID, err)
			}

			finished <- item
		}(<-q.queue)
	}

	for range items {
		select {
		case q.queue <- <-finished:
		case <-time.After(30 * time.Second):
			log.Printf("Error updating queue position: timeout")
			return
		}
	}
}
