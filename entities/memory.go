// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    memory, err := UnmarshalMemory(bytes)
//    bytes, err = memory.Marshal()

package entities

import (
	"encoding/json"
	"github.com/dustin/go-humanize"
)

func UnmarshalMemory(data []byte) (Memory, error) {
	var r Memory
	err := json.Unmarshal(data, &r)
	return r, err
}

func (mem *Memory) Marshal() ([]byte, error) {
	return json.Marshal(mem)
}

type Memory struct {
	RAM  RAM  `json:"ram"`
	Cuda Cuda `json:"cuda"`
}

type Cuda struct {
	System    RAM    `json:"system"`
	Active    Active `json:"active"`
	Allocated Active `json:"allocated"`
	Reserved  Active `json:"reserved"`
	Inactive  Active `json:"inactive"`
	Events    Events `json:"events"`
}

type Active struct {
	Current float64 `json:"current"`
	Peak    float64 `json:"peak"`
}

type Events struct {
	Retries float64 `json:"retries"`
	OOM     float64 `json:"oom"`
}

type RAM struct {
	Free  float64 `json:"free"`
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

type ReadableMemory struct {
	Free  string `json:"free"`
	Used  string `json:"used"`
	Total string `json:"total"`
}

func (mem *RAM) Readable() *ReadableMemory {
	return &ReadableMemory{
		Free:  readableMemory(mem.Free),
		Used:  readableMemory(mem.Used),
		Total: readableMemory(mem.Total),
	}
}

func (mem *Cuda) Readable() *ReadableMemory {
	return mem.System.Readable()
}

func readableMemory(bytes float64) string {
	return humanize.IBytes(uint64(bytes))
}
