package lib

import (
	"errors"
	"hash/fnv"
	"sort"
)

const (
	ToggleEnabled  = true
	ToggleDisabled = true
)

type Toggle struct {
	Name        string
	Description string
	Default     bool
	Rules       []*ToggleRule
}

type ToggleRule struct {
	Type   string
	Params J
}

func ToggleRulePercent(field string, percent int) *ToggleRule {
	return &ToggleRule{
		Type:   "percent",
		Params: J{"percent": percent},
	}
}

func ToggleRuleMatch(field string, ids ...string) *ToggleRule {
	return &ToggleRule{
		Type:   "match",
		Params: J{"field": field, "ids": ids},
	}
}

func (tr *ToggleRule) Evaluate(c *Ctx) bool {
	if tr.Type == "percent" {
		value := c.Data.Get(tr.Params.Get("field"))
		if value != "" {
			h := fnv.New32a()
			h.Write([]byte(value))
			return (h.Sum32()%100)+1 <= uint32(tr.Params["percent"].(int))
		}
		return false
	} else if tr.Type == "match" {
		value := c.Data.Get(tr.Params.Get("field"))
		ids := tr.Params["ids"].([]string)
		i := sort.SearchStrings(ids, value)
		if value != "" && i < len(ids) && ids[i] == value {
			return true
		}
		return false
	} else {
		panic(errors.New("Toggle: Unknown ToggleRule type: " + tr.Type))
	}
}

var globalToggles = map[string]*Toggle{}

func RegisterToggle(t *Toggle) string {
	globalToggles[t.Name] = t
	return t.Name
}

func (c *Ctx) Toggle(name string) bool {
	toggle, ok := globalToggles[name]
	if !ok {
		panic(errors.New("Toggles: Tried to use non existant toggle: " + name))
	}
	for _, r := range toggle.Rules {
		if r.Evaluate(c) {
			return !toggle.Default
		}
	}
	return toggle.Default
}
