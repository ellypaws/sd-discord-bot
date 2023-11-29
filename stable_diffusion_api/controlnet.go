// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    controlnetModules, err := UnmarshalControlnetModules(bytes)
//    bytes, err = controlnetModules.Marshal()

package stable_diffusion_api

import "encoding/json"

func UnmarshalControlnetTypes(data []byte) (ControlnetTypes, error) {
	var r ControlnetTypes
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ControlnetTypes) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ControlnetTypes struct {
	ControlTypes map[string]ControlType `json:"control_types"`
	Modules      *ControlnetModules     `json:"-"` // Deprecated: Do not use this field. Use ControlnetModulesCache instead.
	Models       *ControlnetModels      `json:"-"` // Deprecated: Do not use this field. Use ControlnetModelsCache instead.
}

type ControlType struct {
	ModuleList    []string `json:"module_list"`
	ModelList     []string `json:"model_list"`
	DefaultOption string   `json:"default_option"`
	DefaultModel  string   `json:"default_model"`
}

var ControlnetTypesCache *ControlnetTypes

func AllControlnetTypes(api StableDiffusionAPI) ([]string, ControlnetTypes, ControlnetModules, ControlnetModels) {
	types, _ := ControlnetTypesCache.GetCache(api)
	modules, _ := ControlnetModulesCache.GetCache(api)
	models, _ := ControlnetModelsCache.GetCache(api)
	var typesList []string
	for key := range types.(*ControlnetTypes).ControlTypes {
		typesList = append(typesList, key)
	}
	return typesList, *types.(*ControlnetTypes), *modules.(*ControlnetModules), *models.(*ControlnetModels)
}

// Deprecated: Do not use this method. Use ControlnetModules.String or ControlnetModels.String instead.
func (c ControlnetTypes) String(i int) string {
	return (*c.Models)[i].Type
}

func (c ControlnetTypes) Len() int {
	return len(c.ControlTypes)
}

func (c *ControlnetTypes) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if ControlnetTypesCache != nil {
		return ControlnetTypesCache, nil
	}
	return c.apiGET(api)
}

func (c *ControlnetTypes) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	bytes, err := api.GET("/controlnet/control_types")
	if err != nil {
		return nil, err
	}

	if ControlnetTypesCache == nil {
		ControlnetTypesCache = &ControlnetTypes{}
	}
	err = json.Unmarshal(bytes, ControlnetTypesCache)
	if err != nil {
		return nil, err
	}

	for key, controlType := range ControlnetTypesCache.ControlTypes {
		for _, module := range controlType.ModuleList {
			if ControlnetModulesCache == nil {
				ControlnetModulesCache = &ControlnetModules{}
			}
			*ControlnetModulesCache = append(*ControlnetModulesCache, ControlnetModule{
				Type:    key,
				Module:  module,
				Models:  controlType.ModelList,
				Default: controlType.DefaultOption == module,
			})
		}
		//c.Modules = ControlnetModulesCache
		for _, model := range controlType.ModelList {
			if ControlnetModelsCache == nil {
				ControlnetModelsCache = &ControlnetModels{}
			}
			*ControlnetModelsCache = append(*ControlnetModelsCache, ControlnetModel{
				Type:    key,
				Model:   model,
				Modules: controlType.ModuleList,
				Default: controlType.DefaultModel == model,
			})
		}
		//c.Models = ControlnetModelsCache
	}

	return ControlnetTypesCache, nil
}

func (c *ControlnetTypes) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	// no refresh available
	return c.apiGET(api)
}

type ControlnetModule struct {
	Type    string   `json:"type"`
	Module  string   `json:"module"`
	Models  []string `json:"models"`
	Default bool     `json:"default"`
}

type ControlnetModules []ControlnetModule

func (c ControlnetModules) String(i int) string {
	return c[i].Module
}

func (c ControlnetModules) Len() int {
	return len(c)
}

var ControlnetModulesCache *ControlnetModules

func (c *ControlnetModules) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if ControlnetModulesCache != nil {
		return ControlnetModulesCache, nil
	}
	return c.apiGET(api)
}

func (c *ControlnetModules) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	controlnetTypes, err := ControlnetTypesCache.GetCache(api)
	if err != nil {
		return nil, err
	}
	if controlnetTypes.(*ControlnetTypes).Modules != nil {
		return controlnetTypes.(*ControlnetTypes).Modules, nil
	}
	return ControlnetModulesCache, nil
}

func (c *ControlnetModules) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	// no refresh available
	return c.apiGET(api)
}

type ControlnetModel struct {
	Type    string   `json:"type"`
	Model   string   `json:"model"`
	Modules []string `json:"modules"`
	Default bool     `json:"default"`
}

type ControlnetModels []ControlnetModel

func (c ControlnetModels) String(i int) string {
	return c[i].Model
}

func (c ControlnetModels) Len() int {
	return len(c)
}

var ControlnetModelsCache *ControlnetModels

func (c *ControlnetModels) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if ControlnetModelsCache != nil {
		return ControlnetModelsCache, nil
	}
	return c.apiGET(api)
}

func (c *ControlnetModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	controlnetTypes, err := ControlnetTypesCache.GetCache(api)
	if err != nil {
		return nil, err
	}
	if controlnetTypes.(*ControlnetTypes).Models != nil {
		return controlnetTypes.(*ControlnetTypes).Models, nil
	}
	return ControlnetModelsCache, nil
}

func (c *ControlnetModels) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	// no refresh available
	return c.apiGET(api)
}
