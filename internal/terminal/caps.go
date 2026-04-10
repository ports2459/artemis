package terminal

import (
	"os"
	"strings"
)

type ImageProtocol int

const (
	ProtocolASCII ImageProtocol = iota
	ProtocolSixel
	ProtocolKitty
)

func (p ImageProtocol) String() string {
	switch p {
	case ProtocolKitty:
		return "kitty"
	case ProtocolSixel:
		return "sixel"
	default:
		return "ascii"
	}
}

type Capabilities struct {
	ImageProtocol ImageProtocol
}

func Detect(override string) Capabilities {
	switch strings.ToLower(override) {
	case "kitty":
		return Capabilities{ImageProtocol: ProtocolKitty}
	case "sixel":
		return Capabilities{ImageProtocol: ProtocolSixel}
	case "ascii":
		return Capabilities{ImageProtocol: ProtocolASCII}
	default:
		return detectFromEnv(os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"))
	}
}

func detectFromEnv(termProgram, term string) Capabilities {
	tp := strings.ToLower(termProgram)
	t := strings.ToLower(term)

	if strings.Contains(t, "kitty") || tp == "kitty" {
		return Capabilities{ImageProtocol: ProtocolKitty}
	}

	sixelTerminals := []string{"wezterm", "foot", "contour", "mintty", "mlterm"}
	for _, st := range sixelTerminals {
		if strings.Contains(tp, st) {
			return Capabilities{ImageProtocol: ProtocolSixel}
		}
	}

	if tp == "xterm" {
		return Capabilities{ImageProtocol: ProtocolSixel}
	}

	return Capabilities{ImageProtocol: ProtocolASCII}
}
