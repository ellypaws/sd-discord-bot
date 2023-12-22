package stable_diffusion_api

import "stable_diffusion_bot/entities"

func (api *apiImplementation) GetMemory() (*entities.Memory, error) {
	getURL := "/sdapi/v1/memory"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	memory, err := entities.UnmarshalMemory(body)
	if err != nil {
		return nil, err
	}

	return &memory, nil
}

func (api *apiImplementation) GetMemoryReadable() (*entities.ReadableMemory, error) {
	memory, err := api.GetMemory()
	if err != nil {
		return nil, err
	}

	return memory.RAM.Readable(), nil
}

func (api *apiImplementation) GetVRAMReadable() (*entities.ReadableMemory, error) {
	memory, err := api.GetMemory()
	if err != nil {
		return nil, err
	}

	return memory.Cuda.System.Readable(), nil
}
