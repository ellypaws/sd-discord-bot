package imagine_queue

import (
	"encoding/json"
	"log"
	"stable_diffusion_bot/entities"
	"strconv"
)

func (q *queueImplementation) processCurrentImagine() {
	go func() {
		defer func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			q.currentImagine = nil
		}()

		if q.currentImagine.Type == ItemTypeUpscale {
			q.processUpscaleImagine(q.currentImagine)

			return
		}

		newGeneration, err := &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				Processed:    false,
				Checkpoint:   q.currentImagine.Checkpoint,
				VAE:          q.currentImagine.VAE,
				Hypernetwork: q.currentImagine.Hypernetwork,
			},
			TextToImageRequest: &entities.TextToImageRequest{
				Prompt:            q.currentImagine.Prompt,
				NegativePrompt:    q.currentImagine.NegativePrompt,
				Width:             initializedWidth,
				Height:            initializedHeight,
				RestoreFaces:      q.currentImagine.RestoreFaces,
				EnableHr:          q.currentImagine.UseHiresFix,
				HrScale:           between(q.currentImagine.HiresUpscaleRate, 1.0, 2.0),
				HrUpscaler:        "R-ESRGAN 2x+",
				HrSecondPassSteps: q.currentImagine.HiresSteps,
				HrResizeX:         initializedWidth,
				HrResizeY:         initializedHeight,
				DenoisingStrength: q.currentImagine.DenoisingStrength,
				Seed:              q.currentImagine.Seed,
				Subseed:           -1,
				SubseedStrength:   0,
				SamplerName:       q.currentImagine.SamplerName1,
				CFGScale:          q.currentImagine.CfgScale,
				Steps:             q.currentImagine.Steps,
			},
		}, error(nil)

		newGeneration.Width, err = q.defaultWidth()
		if err != nil {
			log.Printf("Error getting default width: %v", err)
		}

		newGeneration.Height, err = q.defaultHeight()
		if err != nil {
			log.Printf("Error getting default height: %v", err)
		}

		// add optional parameter: Negative prompt
		if q.currentImagine.NegativePrompt == "" {
			newGeneration.NegativePrompt = defaultNegative
		}

		// add optional parameter: sampler
		if q.currentImagine.SamplerName1 == "" {
			newGeneration.SamplerName = "Euler a"
		}

		// extract key value pairs from prompt
		var parameters map[string]string
		parameters, newGeneration.Prompt = extractKeyValuePairsFromPrompt(newGeneration.Prompt)

		defaultWidth := newGeneration.Width
		defaultHeight := newGeneration.Height
		if q.currentImagine.AspectRatio != "" && q.currentImagine.AspectRatio != "1:1" {
			newGeneration.Width, newGeneration.Height = aspectRatioCalculation(q.currentImagine.AspectRatio, defaultWidth, defaultHeight)
		} else {
			if aspectRatio, ok := parameters["ar"]; ok {
				newGeneration.Width, newGeneration.Height = aspectRatioCalculation(aspectRatio, defaultWidth, defaultHeight)
			}
		}

		// extract --zoom parameter
		adjustedWidth := newGeneration.Width
		adjustedHeight := newGeneration.Height
		if newGeneration.EnableHr && newGeneration.HrScale > 1.0 {
			newGeneration.HrResizeX = int(float64(adjustedWidth) * newGeneration.HrScale)
			newGeneration.HrResizeY = int(float64(adjustedHeight) * newGeneration.HrScale)
		} else {
			newGeneration.EnableHr = false
			newGeneration.HrResizeX = adjustedWidth
			newGeneration.HrResizeY = adjustedHeight
		}

		if zoom, ok := parameters["zoom"]; ok {
			zoomScale, err := strconv.ParseFloat(zoom, 64)
			if err != nil {
				log.Printf("Error extracting zoom scale from prompt: %v", err)
			} else {
				newGeneration.EnableHr = true
				newGeneration.HrScale = between(zoomScale, 1.0, 2.0)
				newGeneration.HrResizeX = int(float64(adjustedWidth) * newGeneration.HrScale)
				newGeneration.HrResizeY = int(float64(adjustedHeight) * newGeneration.HrScale)
			}
		}

		if step, ok := parameters["step"]; ok {
			stepInt, err := strconv.Atoi(step)
			if err != nil {
				log.Printf("Error extracting step from prompt: %v", err)
			} else {
				newGeneration.Steps = stepInt
			}
		}

		if cfgscale, ok := parameters["cfgscale"]; ok {
			cfgScaleFloat, err := strconv.ParseFloat(cfgscale, 64)
			if err != nil {
				log.Printf("Error extracting cfg scale from prompt: %v", err)
			} else {
				newGeneration.CFGScale = cfgScaleFloat
			}
		}

		if seed, ok := parameters["seed"]; ok {
			seedInt, err := strconv.ParseInt(seed, 10, 64)
			if err != nil {
				log.Printf("Error extracting seed from prompt: %v", err)
			} else {
				newGeneration.Seed = seedInt
			}
		}

		// prompt will display as Monospace in Discord
		//var quotedPrompt = quotePromptAsMonospace(promptRes4.SanitizedPrompt)
		//promptRes.SanitizedPrompt = quotedPrompt

		config, err := q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
		} else {
			if !ptrStringNotBlank(newGeneration.Checkpoint) {
				newGeneration.Checkpoint = config.SDModelCheckpoint
			}
			if !ptrStringNotBlank(newGeneration.VAE) {
				newGeneration.VAE = config.SDVae
			}
			if !ptrStringNotBlank(newGeneration.Hypernetwork) {
				newGeneration.Hypernetwork = config.SDHypernetwork
			}
		}

		// segModelOptions will never be nil and at least an empty string in the slice [""]
		// because of strings.Split() in discord_bot.go

		//additionalScript := make(map[string]*entities.ADetailer)
		//alternatively additionalScript := map[string]*stable_diffusion_api.ADetailer{}

		if q.currentImagine.ADetailerString != "" {
			log.Printf("q.currentImagine.ADetailerString: %v", q.currentImagine.ADetailerString)

			newGeneration.NewADetailer()

			newGeneration.AlwaysonScripts.ADetailer.AppendSegModelByString(q.currentImagine.ADetailerString, newGeneration)
		}

		if newGeneration.AlwaysonScripts != nil {
			jsonMarshalScripts, err := json.MarshalIndent(&newGeneration.AlwaysonScripts, "", "  ")
			if err != nil {
				log.Printf("Error marshalling scripts: %v", err)
			} else {
				log.Println("Final scripts: ", string(jsonMarshalScripts))
			}
		}

		// Should not create a new map here, because it will be overwritten by the map in newGeneration
		// if newGeneration.AlwaysOnScripts == nil {
		// 	newGeneration.AlwaysOnScripts = make(map[string]*entities.ADetailer)
		// }

		//if additionalScript["ADetailer"] != nil {
		//	newGeneration.AlwaysOnScripts["ADetailer"] = additionalScript["ADetailer"]
		//}

		switch q.currentImagine.Type {
		case ItemTypeReroll, ItemTypeVariation:
			foundGeneration, err := q.getPreviousGeneration(q.currentImagine, q.currentImagine.InteractionIndex)
			if err != nil {
				log.Printf("Error getting prompt for reroll: %v", err)

				return
			}

			// if we are rerolling, or generating variations, we simply replace some defaults
			newGeneration = foundGeneration

			// for variations, we need random subseeds
			newGeneration.Subseed = -1

			// for variations, the subseed strength determines how much variation we get
			if q.currentImagine.Type == ItemTypeVariation {
				newGeneration.SubseedStrength = 0.15
			}
		}

		err = q.processImagineGrid(newGeneration, q.currentImagine)
		if err != nil {
			log.Printf("Error processing imagine grid: %v", err)

			return
		}
	}()
}
