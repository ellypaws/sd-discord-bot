package novelai

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
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
		q.current = <-q.queue
		requireInteraction(q.current.DiscordInteraction)

		if q.cancelled[q.current.DiscordInteraction.ID] {
			// If the item is cancelled, skip it
			delete(q.cancelled, q.current.DiscordInteraction.ID)
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
	return nil
}

func requireInteraction(i *discordgo.Interaction) {
	if i != nil {
		return
	}
	log.Panicf("Interaction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", i)
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
	removed := 1

	for range items {
		item := <-q.queue
		if q.cancelled[item.DiscordInteraction.ID] {
			delete(q.cancelled, item.DiscordInteraction.ID)
			removed++
			continue
		}
		item.pos = item.pos - removed
		finished <- item

		go func(item *NAIQueueItem) {
			_, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, q.positionString(item), handlers.Components[handlers.Cancel])
			if err != nil {
				log.Printf("Error updating queue position for item %v: %v", item.DiscordInteraction.ID, err)
			}
		}(item)
	}

	timeout := time.NewTimer(30 * time.Second)
	defer drain(timeout)
	for range items {
		select {
		case q.queue <- <-finished:
		case <-timeout.C:
			log.Printf("Error updating queue position: timeout")
			return
		}
	}
}

func drain(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}
