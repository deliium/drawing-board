package curriculum

import (
	contenthiragana5 "github.com/deliium/drawing-board/content/hiragana5"
)

const EmbeddedV1Dir = "v1"

// LoadEmbeddedV1 loads the embedded content/hiragana5/v1 pack and validates it.
func LoadEmbeddedV1() (Pack, error) {
	return LoadPack(contenthiragana5.V1, EmbeddedV1Dir)
}

// LoadPublishedV1 loads the embedded pack and requires reviewStatus=published.
func LoadPublishedV1() (Pack, error) {
	p, err := LoadEmbeddedV1()
	if err != nil {
		return Pack{}, err
	}
	if err := RequirePublished(p); err != nil {
		logf("ERROR", "[curriculum.load] %v", err)
		return Pack{}, err
	}
	return p, nil
}
