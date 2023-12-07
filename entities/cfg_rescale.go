package entities

import (
	"encoding/json"
	"errors"
)

type CFGRescaleParameters struct {
	CfgRescale   float64 //`json:"1,omitempty"`
	AutoColorFix bool    //`json:"2,omitempty"`
	FixStrength  float64 //`json:"3,omitempty"`
	KeepOriginal bool    //`json:"4,omitempty"`
}

func (p CFGRescaleParameters) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{p.CfgRescale, p.AutoColorFix, p.FixStrength, p.KeepOriginal})
}

func (p *CFGRescaleParameters) UnmarshalJSON(data []byte) error {
	var a []any
	err := json.Unmarshal(data, &a)
	if err != nil {
		return err
	}

	for i, v := range a {
		var ok bool
		switch i {
		case 0:
			p.CfgRescale, ok = v.(float64)
			if !ok {
				return errors.New("expected float64 for CfgRescale")
			}

		case 1:
			p.AutoColorFix, ok = v.(bool)
			if !ok {
				return errors.New("expected bool for AutoColorFix")
			}

		case 2:
			p.FixStrength, ok = v.(float64)
			if !ok {
				return errors.New("expected float64 for FixStrength")
			}

		case 3:
			p.KeepOriginal, ok = v.(bool)
			if !ok {
				return errors.New("expected bool for KeepOriginal")
			}
		}
	}
	return nil
}

type CFGRescale struct {
	Args CFGRescaleParameters `json:"args,omitempty"`
}
