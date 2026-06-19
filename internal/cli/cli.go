// Package cli wires yoink's subcommands together and is the single entry point
// invoked from main.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/davcevski/yoink/internal/buildinfo"
	"github.com/davcevski/yoink/internal/clipboard"
	"github.com/davcevski/yoink/internal/config"
	"github.com/davcevski/yoink/internal/crypto"
	"github.com/davcevski/yoink/internal/daemon"
	"github.com/davcevski/yoink/internal/keyring"
	"github.com/davcevski/yoink/internal/launchd"
	"github.com/davcevski/yoink/internal/paths"
	"github.com/davcevski/yoink/internal/store"
	"github.com/davcevski/yoink/internal/tui"
)

// Run dispatches a subcommand. args is os.Args[1:].
func Run(args []string) error {
	if len(args) == 0 {
		return runTUI(nil)
	}
	switch args[0] {
	case "daemon":
		return runDaemon()
	case "install":
		return runInstall()
	case "uninstall":
		return runUninstall()
	case "search":
		return runSearch(args[1:])
	case "copy":
		return runCopy(args[1:])
	case "config":
		return runConfig(args[1:])
	case "version", "--version", "-v":
		fmt.Println("yoink " + buildinfo.String())
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		if strings.HasPrefix(args[0], "-") {
			return runTUI(args) // e.g. `yoink --limit 10`
		}
		return fmt.Errorf("unknown command %q (try `yoink help`)", args[0])
	}
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("yoink", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "number of clips to display (default: config display_limit)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, _, err := config.LoadOrCreate()
	if err != nil {
		return err
	}
	displayLimit := cfg.DisplayLimit
	if *limit > 0 {
		displayLimit = *limit
	}

	cipher, zero, err := openCipher(keyring.New())
	if err != nil {
		return err
	}
	defer zero()

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	// Load enough clips to support cycling the display limit up to 20 in-TUI.
	loadLimit := displayLimit
	if loadLimit < 20 {
		loadLimit = 20
	}
	clips, err := db.Recent(loadLimit)
	if err != nil {
		return err
	}

	model := tui.New(tui.Decrypt(clips, cipher), clipboard.New(), db, displayLimit)
	final, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	if m, ok := final.(tui.Model); ok {
		if s := m.Status(); s != "" {
			fmt.Println(s)
		}
	}
	return nil
}

func runDaemon() error {
	cfg, _, err := config.LoadOrCreate()
	if err != nil {
		return err
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	// launchd redirects stdout/stderr to ~/.config/yoink/yoinkd.log.
	logger := log.New(os.Stderr, "", log.LstdFlags)
	d := daemon.New(clipboard.New(), db, keyring.New(), cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return d.Run(ctx)
}

func runInstall() error {
	if _, _, err := config.LoadOrCreate(); err != nil {
		return err
	}
	// Provision the encryption key now (the daemon never generates one).
	if _, err := keyring.EnsureKey(keyring.New()); err != nil {
		return fmt.Errorf("provisioning encryption key: %w", err)
	}

	bin, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	logPath, err := paths.LogFile()
	if err != nil {
		return err
	}
	if err := launchd.Install(bin, logPath); err != nil {
		return err
	}

	fmt.Println("✓ yoink daemon installed and started (runs at login).")
	fmt.Printf("  logs: %s\n", logPath)

	if on, err := launchd.FileVaultEnabled(); err == nil && !on {
		fmt.Println()
		fmt.Println("⚠ FileVault is OFF. yoink encrypts clips at rest, but enabling")
		fmt.Println("  FileVault protects the rest of your disk too. Turn it on in")
		fmt.Println("  System Settings → Privacy & Security → FileVault.")
	}
	return nil
}

func runUninstall() error {
	if err := launchd.Uninstall(); err != nil {
		return err
	}
	fmt.Println("✓ yoink daemon uninstalled.")
	return nil
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	edit := fs.Bool("edit", false, "open the config file in $EDITOR")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, path, err := config.LoadOrCreate()
	if err != nil {
		return err
	}

	if *edit {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}

	fmt.Println("# " + path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

// runSearch prints clips matching a query without launching the TUI, for use by
// launchers (Raycast/Alfred) and scripts. Output is "<id>\t<preview>" lines, or
// a JSON array with --json.
func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "output results as JSON")
	limit := fs.Int("limit", 50, "maximum number of results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")

	cipher, zero, err := openCipher(keyring.New())
	if err != nil {
		return err
	}
	defer zero()

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	// Load a generous window so fuzzy matches further back are still found.
	loadLimit := *limit * 4
	if loadLimit < 200 {
		loadLimit = 200
	}
	clips, err := db.Recent(loadLimit)
	if err != nil {
		return err
	}
	results := tui.Search(tui.Decrypt(clips, cipher), query, *limit)

	if *asJSON {
		type row struct {
			ID      int64  `json:"id"`
			Preview string `json:"preview"`
			Bytes   int    `json:"bytes"`
			Created int64  `json:"created_at"`
		}
		out := make([]row, len(results))
		for i, it := range results {
			out[i] = row{it.ID, tui.Preview(it.Text, 120), it.Bytes, it.CreatedAt.Unix()}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	for _, it := range results {
		fmt.Printf("%d\t%s\n", it.ID, tui.Preview(it.Text, 120))
	}
	return nil
}

// runCopy writes the clip with the given id back to the clipboard without the
// TUI — the non-interactive "yoink it back" used by launcher actions.
func runCopy(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: yoink copy <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid clip id %q", args[0])
	}

	cipher, zero, err := openCipher(keyring.New())
	if err != nil {
		return err
	}
	defer zero()

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	clip, err := db.Get(id)
	if err != nil {
		return fmt.Errorf("clip %d not found", id)
	}
	plaintext, err := cipher.Open(clip.Content, clip.Nonce)
	if err != nil {
		return fmt.Errorf("decrypting clip %d: %w", id, err)
	}
	if err := clipboard.New().Write(string(plaintext)); err != nil {
		return err
	}
	fmt.Printf("yoinked clip %d (%d bytes) to the clipboard\n", id, clip.Bytes)
	return nil
}

// openCipher loads the encryption key (creating it if absent) and returns a
// Cipher plus a cleanup that zeroes the key material.
func openCipher(k keyring.Keyring) (*crypto.Cipher, func(), error) {
	key, err := keyring.EnsureKey(k)
	if err != nil {
		return nil, nil, fmt.Errorf("reading encryption key: %w", err)
	}
	cipher, err := crypto.New(key)
	if err != nil {
		return nil, nil, err
	}
	return cipher, cipher.Zero, nil
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `yoink — an encrypted macOS clipboard history

usage:
  yoink                 launch the clipboard browser (TUI)
  yoink --limit N       launch the TUI showing N clips
  yoink search [query]  print matching clips as "<id>\t<preview>" (--json too)
  yoink copy <id>       copy a clip back to the clipboard by id (no TUI)
  yoink daemon          run the capture/prune loop (used by the LaunchAgent)
  yoink install         install + start the background daemon (run once)
  yoink uninstall       stop + remove the background daemon
  yoink config          print the config file path and contents
  yoink config --edit   open the config file in $EDITOR
  yoink version         print version information
  yoink help            show this help

TUI keys:
  ↑/↓ or j/k  move        enter  yoink back     d  delete
  /           search      < / >  display limit  q  quit
  (just start typing to search incrementally)
`)
}
