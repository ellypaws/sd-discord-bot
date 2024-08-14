package novelai

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"sync"
	"time"
)

func (q *NAIQueue) next() error {
	if len(q.queue) == 0 {
		return nil
	}

	if q.current != nil {
		log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
		return fmt.Errorf("currentImagine is not nil")
	}
	q.current = <-q.queue
	defer q.done()
	requireInteraction(q.current.DiscordInteraction)

	q.mu.Lock()
	if q.cancelled[q.current.DiscordInteraction.ID] {
		// If the item is cancelled, skip it
		delete(q.cancelled, q.current.DiscordInteraction.ID)
		q.mu.Unlock()
		return nil
	}
	q.mu.Unlock()

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
		return handlers.ErrorEdit(q.botSession, q.current.DiscordInteraction, fmt.Errorf("unknown item type: %s", q.current.Type))
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

	if items == 0 {
		return
	}

	finished := make(chan *NAIQueueItem, items)

	var position int
	var updated sync.WaitGroup
	for range items {
		item := <-q.queue
		if q.cancelled[item.DiscordInteraction.ID] {
			delete(q.cancelled, item.DiscordInteraction.ID)
			continue
		}
		item.pos = position
		position++
		finished <- item

		updated.Add(1)
		go func(item *NAIQueueItem) {
			_, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, q.positionString(item), handlers.Components[handlers.Cancel])
			if err != nil {
				log.Printf("Error updating queue position for item %v: %v", item.DiscordInteraction.ID, err)
			}
			updated.Done()
		}(item)
	}
	updated.Wait()

	timeout := time.NewTimer(30 * time.Second)
	for range items {
		select {
		case q.queue <- <-finished:
		case <-timeout.C:
			log.Printf("Error updating queue position: timeout")
			return
		}
	}

	drain(timeout)
}

func drain(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}
