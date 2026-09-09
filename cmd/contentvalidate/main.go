package main

import (
	"fmt"
	"os"

	contenthiragana5 "github.com/deliium/drawing-board/content/hiragana5"
	"github.com/deliium/drawing-board/internal/curriculum"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-hash" {
		// Bypass published/hash gate: load raw files and print ContentHash.
		raw, err := loadRawForHash()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR [contentvalidate] %v\n", err)
			os.Exit(1)
		}
		fmt.Println(raw)
		return
	}

	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR [contentvalidate] %v\n", err)
		os.Exit(1)
	}
	audioN := 0
	for _, c := range p.Chars {
		if c.Pronunciation.AudioRef != nil && *c.Pronunciation.AudioRef != "" {
			audioN++
		}
	}
	fmt.Printf("INFO [contentvalidate] ok contentVersion=%s hash=%s characters=%d audioFiles=%d schemaVersion=%d\n",
		p.Manifest.ContentVersion, p.Manifest.ContentHash, len(p.Chars), audioN, p.Manifest.SchemaVersion)
}

func loadRawForHash() (string, error) {
	// Temporarily tolerate placeholder by reading via LoadPack after patching is awkward;
	// duplicate minimal read using exported helpers on embedded FS.
	p, err := curriculum.LoadPackAllowPlaceholder(contenthiragana5.V1, "v1")
	if err != nil {
		return "", err
	}
	return curriculum.ContentHash(p)
}
