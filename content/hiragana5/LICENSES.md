# Licensing notes — hiragana5 content

Root repository license is **CC0 1.0** (`LICENSE` at repo root).

| Asset | Status |
|-------|--------|
| Canonical stroke polylines (`v1/strokes.json`) | Project-authored under CC0 |
| Trace templates (`v1/traces.json`) | Project-authored denser UI paths under CC0 (same stroke count/order as assessment; not a byte-identical alias of `strokes.json`) |
| Character pedagogy copy (descriptions, guidance, example glosses) | Original short glosses under CC0; not copied from textbooks |
| Display fonts (Noto Sans JP / similar) | Self-hosted OFL subsets under `web/public/fonts/` — see font license files there |
| Pronunciation audio (`v1/audio/*.mp3`) | See table below |

Third-party stroke fonts or paths must not enter `v1/` without an explicit compatible license entry here plus attribution.

## Pronunciation audio (Prompt 17)

Primary format: **MP3** (LAME), short isolated mora (~0.85s), loudness-normalized. Canonical files live under `content/hiragana5/v1/audio/`; Vite/dev/production static mirror is `web/public/audio/hiragana5/` (`make sync-audio`).

`audioRef` values are static URL paths such as `/audio/hiragana5/a.mp3` (same path in pack JSON and on the static root).

| File | Source | Author | License | Notes |
|------|--------|--------|---------|-------|
| `a.mp3` | Wikimedia Commons [Ja-A.oga](https://commons.wikimedia.org/wiki/File:Ja-A.oga) | Hakatanoshio117117 | Public domain (author release) | Trimmed + loudness-normalized; human listen pass 2026-09-09 |
| `i.mp3` | Wikimedia Commons [Japanese I.ogg](https://commons.wikimedia.org/wiki/File:Japanese_I.ogg) | Hakatanoshio117117 | Public domain (author release) | Same processing |
| `u.mp3` | Wikimedia Commons [Japanese U.ogg](https://commons.wikimedia.org/wiki/File:Japanese_U.ogg) | Hakatanoshio117117 | Public domain (author release) | Same processing |
| `e.mp3` | Wikimedia Commons [Ja-E.oga](https://commons.wikimedia.org/wiki/File:Ja-E.oga) | Hakatanoshio117117 | Public domain (author release) | Same processing |
| `o.mp3` | Wikimedia Commons [Japanese O.ogg](https://commons.wikimedia.org/wiki/File:Japanese_O.ogg) | Hakatanoshio117117 | Public domain (author release) | Same processing |

These are **human-recorded** native-speaker clips (not TTS). Do not market them as “AI pronunciation,” “perfect native TTS,” or calibrated speech-quality scores. Teaching-quality review is recorded in `v1/review.json` (`audioHumanListenPass`, `audioLicenseRecorded`).

TTS may not be the sole trusted production source unless a human listen pass and redistribution license are documented here and in `review.json`.
