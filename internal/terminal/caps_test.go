package terminal

import "testing"

func TestDetectImageProtocol_Override(t *testing.T) {
	caps := Detect("kitty")
	if caps.ImageProtocol != ProtocolKitty {
		t.Errorf("expected kitty, got %v", caps.ImageProtocol)
	}
}

func TestDetectImageProtocol_AutoWithEnv(t *testing.T) {
	caps := detectFromEnv("WezTerm", "")
	if caps.ImageProtocol != ProtocolSixel {
		t.Errorf("expected sixel for WezTerm, got %v", caps.ImageProtocol)
	}

	caps = detectFromEnv("", "xterm-kitty")
	if caps.ImageProtocol != ProtocolKitty {
		t.Errorf("expected kitty for xterm-kitty, got %v", caps.ImageProtocol)
	}
}

func TestDetectImageProtocol_Fallback(t *testing.T) {
	caps := detectFromEnv("", "xterm-256color")
	if caps.ImageProtocol != ProtocolASCII {
		t.Errorf("expected ASCII fallback, got %v", caps.ImageProtocol)
	}
}
