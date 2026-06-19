package launchd

import (
	"strings"
	"testing"
)

func TestRenderPlistContainsEssentials(t *testing.T) {
	out, err := RenderPlist("/usr/local/bin/yoink", "/Users/x/.config/yoink/yoinkd.log")
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}

	wants := []string{
		"<string>" + Label + "</string>",
		"<string>/usr/local/bin/yoink</string>",
		"<string>daemon</string>",
		"<string>/Users/x/.config/yoink/yoinkd.log</string>",
		"<key>RunAtLoad</key>",
		"<key>KeepAlive</key>",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("plist missing %q\n---\n%s", w, out)
		}
	}
}

func TestRenderPlistIsWellFormedHeader(t *testing.T) {
	out, _ := RenderPlist("/bin/yoink", "/tmp/log")
	if !strings.HasPrefix(out, "<?xml version=\"1.0\"") {
		t.Error("plist missing XML declaration")
	}
	if !strings.Contains(out, "<plist version=\"1.0\">") {
		t.Error("plist missing root element")
	}
}
