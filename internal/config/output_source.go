package config

import "fmt"

type OutputSource int

const (
	OutputSourceCLI OutputSource = iota
	OutputSourceKafka
)

var outputSourceStrings = [...]string{
	"cli",
	"kafka",
}

func (os OutputSource) String() string {
	return outputSourceStrings[os]
}

func ParseOutputSource(s string) (OutputSource, error) {
	for i, v := range outputSourceStrings {
		if s == v {
			return OutputSource(i), nil
		}
	}
	return OutputSourceCLI, fmt.Errorf("unknown OutputSource string: %s", s)
}
