package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/langgenius/tokener-cli/internal/rxsnapshot"
)

const defaultManifest = "internal/agent/rx.lock.json"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: rxmanifest <verify|write>")
	}
	switch args[0] {
	case "verify":
		return verify(args[1:])
	case "write":
		return write(args[1:])
	default:
		return fmt.Errorf("unknown rxmanifest command %q", args[0])
	}
}

func verify(args []string) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	root := flags.String("root", ".", "")
	manifest := flags.String("manifest", defaultManifest, "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("verify accepts no positional arguments")
	}
	snapshot, err := rxsnapshot.Read(filepath.Join(*root, filepath.FromSlash(*manifest)))
	if err != nil {
		return err
	}
	if err := snapshot.VerifyFiles(*root); err != nil {
		return err
	}
	fmt.Printf("rx snapshot %s@%s verified\n", snapshot.Source.Repository, snapshot.Source.Revision)
	return nil
}

func write(args []string) error {
	flags := flag.NewFlagSet("write", flag.ContinueOnError)
	root := flags.String("root", ".", "")
	manifest := flags.String("manifest", defaultManifest, "")
	repository := flags.String("repository", "", "")
	sourceRef := flags.String("source-ref", "", "")
	revision := flags.String("revision", "", "")
	version := flags.String("version", "", "")
	rustToolchain := flags.String("rust-toolchain", "", "")
	provenance := flags.String("provenance", "", "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("write accepts no positional arguments")
	}
	snapshot, err := rxsnapshot.New(
		*root,
		rxsnapshot.Source{
			Repository: *repository,
			Ref:        *sourceRef,
			Revision:   *revision,
			Version:    *version,
		},
		rxsnapshot.Build{
			RustToolchain: *rustToolchain,
			Provenance:    *provenance,
		},
	)
	if err != nil {
		return err
	}
	path := filepath.Join(*root, filepath.FromSlash(*manifest))
	if err := rxsnapshot.Write(path, snapshot); err != nil {
		return err
	}
	fmt.Printf("rx snapshot %s@%s written\n", snapshot.Source.Repository, snapshot.Source.Revision)
	return nil
}
