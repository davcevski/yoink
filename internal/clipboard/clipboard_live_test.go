//go:build darwin

package clipboard_test

import (
	"os"
	"testing"

	"github.com/davcevski/yoink/internal/clipboard"
)

// TestLiveClipboardRoundTrip exercises the real NSPasteboard binding. It is the
// "native boundary" the design leaves out of the normal suite, so it is opt-in:
//
//	YOINK_LIVE_CLIPBOARD=1 go test ./internal/clipboard/
//
// It briefly overwrites and then restores the system clipboard.
func TestLiveClipboardRoundTrip(t *testing.T) {
	if os.Getenv("YOINK_LIVE_CLIPBOARD") == "" {
		t.Skip("set YOINK_LIVE_CLIPBOARD=1 to run the live NSPasteboard test")
	}

	pb := clipboard.New()
	prev, _, hadPrev := pb.Read()
	t.Cleanup(func() {
		if hadPrev {
			_ = pb.Write(prev)
		}
	})

	before := pb.ChangeCount()

	const want = "yoink-live-roundtrip"
	if err := pb.Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, concealed, ok := pb.Read()
	if !ok {
		t.Fatal("Read reported no text after Write")
	}
	if got != want {
		t.Fatalf("Read = %q, want %q", got, want)
	}
	if concealed {
		t.Error("a plain write was reported as concealed")
	}
	if after := pb.ChangeCount(); after <= before {
		t.Errorf("changeCount did not advance: before=%d after=%d", before, after)
	}
}
