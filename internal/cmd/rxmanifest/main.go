package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

const (
	defaultManifest   = "internal/agent/rx.lock.json"
	defaultRepository = "samzong/Recall"
	defaultToolchain  = "stable"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: rxmanifest <verify|write|pull>")
	}
	command := args[0]
	switch command {
	case "verify", "write", "pull":
	default:
		return fmt.Errorf("unknown rxmanifest command %q", command)
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	root := flags.String("root", ".", "")
	manifest := flags.String("manifest", defaultManifest, "")
	var source rxsnapshot.Source
	var build rxsnapshot.Build
	var tag string
	switch command {
	case "write":
		flags.StringVar(&source.Repository, "repository", "", "")
		flags.StringVar(&source.Ref, "source-ref", "", "")
		flags.StringVar(&source.Revision, "revision", "", "")
		flags.StringVar(&source.Version, "version", "", "")
		flags.StringVar(&build.RustToolchain, "rust-toolchain", "", "")
		flags.StringVar(&build.Provenance, "provenance", "", "")
	case "pull":
		flags.StringVar(&source.Repository, "repository", defaultRepository, "")
		flags.StringVar(&tag, "tag", "", "")
		flags.StringVar(&build.RustToolchain, "rust-toolchain", defaultToolchain, "")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%s accepts no positional arguments", command)
	}
	path := filepath.Join(*root, filepath.FromSlash(*manifest))
	if command == "pull" {
		if strings.TrimSpace(tag) == "" {
			return errors.New("pull requires -tag")
		}
		return pull(*root, path, source.Repository, tag, build.RustToolchain)
	}
	if command == "verify" {
		snapshot, err := rxsnapshot.Read(path)
		if err != nil {
			return err
		}
		if err := snapshot.VerifyFiles(*root); err != nil {
			return err
		}
		fmt.Printf("rx snapshot %s@%s verified\n", snapshot.Source.Repository, snapshot.Source.Revision)
		return nil
	}
	snapshot, err := rxsnapshot.New(*root, source, build)
	if err != nil {
		return err
	}
	if err := rxsnapshot.Write(path, snapshot); err != nil {
		return err
	}
	fmt.Printf("rx snapshot %s@%s written\n", snapshot.Source.Repository, snapshot.Source.Revision)
	return nil
}
