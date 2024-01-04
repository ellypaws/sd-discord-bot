package stable_diffusion_api

import (
	"github.com/shirou/gopsutil/mem"
	"stable_diffusion_bot/entities"
)

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

// GetMemory returns the current memory usage of the system and the GPU as returned by the system, not the API.
func GetMemory() (*entities.Memory, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	memory := &entities.Memory{
		RAM: entities.RAM{
			Free:  float64(vmem.Free),
			Used:  float64(vmem.Used),
			Total: float64(vmem.Total),
		},
		// Cuda memory information is not available from gopsutil
	}

	return memory, nil
}

func GetMemoryReadable() (*entities.ReadableMemory, error) {
	memory, err := GetMemory()
	if err != nil {
		return nil, err
	}

	return memory.RAM.Readable(), nil
}
