package config

import (
	"fmt"
)

type EvictionStrategy int

const (
	EvictionStrategyUnknown EvictionStrategy = iota
	EvictionStrategyLRU
	EvictionStrategyLFU
)

var evictionStrategyStrings = [...]string{
	"LRU",
	"LFU",
}

func (es *EvictionStrategy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	strategy, err := ParseEvictionStrategy(s)
	if err != nil {
		return err
	}
	*es = strategy
	return nil
}

func (es EvictionStrategy) String() string {
	if int(es) < len(evictionStrategyStrings) {
		return evictionStrategyStrings[es]
	}
	return fmt.Sprintf("unknown EvictionStrategy(%d)", es)
}

func ParseEvictionStrategy(s string) (EvictionStrategy, error) {
	for i, v := range evictionStrategyStrings {
		if s == v {
			return EvictionStrategy(i), nil
		}
	}
	return EvictionStrategyUnknown, fmt.Errorf("unknown EvictionStrategy string: %s", s)
}
