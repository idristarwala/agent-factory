package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/llm"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// RecordEvidence holds the inputs shared by the worker and synchronous judge.
type RecordEvidence struct {
	Text   []string
	Images []llm.ImagePart
}

func (r ContextRecipe) collect(ctx context.Context, result map[string]any, client *http.Client) (RecordEvidence, error) {
	out := RecordEvidence{Text: r.Evidence(result)}
	if r.Media == "" {
		return out, nil
	}
	body, ok := result["entity"].(string)
	if !ok {
		return RecordEvidence{}, fmt.Errorf("media requires entity evidence")
	}
	var record map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &record); err != nil {
		return RecordEvidence{}, fmt.Errorf("media source is not a JSON object")
	}
	var urls []string
	field, ok := record[r.Media]
	if !ok || string(field) == "null" || json.Unmarshal(field, &urls) != nil {
		return RecordEvidence{}, fmt.Errorf("media field %q requires an array of image URLs", r.Media)
	}
	return collectImages(ctx, client, urls, out, func(i int) string {
		return fmt.Sprintf("image %d (tool %s, field %s)", i, r.Entity, r.Media)
	})
}

func readRecordImage(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {

	if validateDestination(url, url) != nil {
		return nil, "", fmt.Errorf("invalid image destination")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("image request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image source returned status %d", resp.StatusCode)
	}
	const limit = 512 << 10
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(b) > limit {
		return nil, "", fmt.Errorf("unable to read image within 512 KiB limit")
	}
	mime := http.DetectContentType(b)
	switch mime {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return nil, "", fmt.Errorf("unsupported image content")
	}
	return b, mime, nil
}

func (r ContextRecipe) Collect(ctx context.Context, result map[string]any) (RecordEvidence, error) {
	if r.Media == "" {
		return RecordEvidence{Text: r.Evidence(result)}, nil
	}
	client := publicClient()
	defer client.CloseIdleConnections()
	return r.collect(ctx, result, client)
}

// CollectMappedImages uses the same bounded public transport as tool-provided media.
func CollectMappedImages(ctx context.Context, urls []string) (RecordEvidence, error) {
	client := publicClient()
	defer client.CloseIdleConnections()
	return collectImages(ctx, client, urls, RecordEvidence{}, func(i int) string { return fmt.Sprintf("mapped image %d", i) })
}

func collectImages(ctx context.Context, client *http.Client, urls []string, out RecordEvidence, label func(int) string) (RecordEvidence, error) {
	if len(urls) > llm.MaxRecordImages {
		return RecordEvidence{}, fmt.Errorf("maximum %d images per record", llm.MaxRecordImages)
	}
	const maxParts = 10
	const cell = 320
	perSheet := max(1, (len(urls)+maxParts-1)/maxParts)
	for start := 0; start < len(urls); start += perSheet {
		end := min(start+perSheet, len(urls))
		cols := 1
		for cols*cols < end-start {
			cols++
		}
		var sheet *image.RGBA
		if perSheet > 1 {
			rows := (end - start + cols - 1) / cols
			sheet = image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
			draw.Draw(sheet, sheet.Bounds(), image.White, image.Point{}, draw.Src)
		}
		for index := start; index < end; index++ {
			raw, mime, err := readRecordImage(ctx, client, urls[index])
			if err != nil {
				return RecordEvidence{}, fmt.Errorf("%s: %w", label(index+1), err)
			}
			note := fmt.Sprintf("%s: sha256 %x", label(index+1), sha256.Sum256(raw))
			if sheet == nil {
				out.Images = append(out.Images, llm.ImagePart{Mime: mime, Base64: base64.StdEncoding.EncodeToString(raw)})
			} else {
				row, col := (index-start)/cols, (index-start)%cols
				if err := drawTile(sheet, raw, image.Pt(col*cell, row*cell), cell); err != nil {
					return RecordEvidence{}, fmt.Errorf("%s: %w", label(index+1), err)
				}
				note += fmt.Sprintf(" (sheet %d, row %d, column %d)", len(out.Images)+1, row+1, col+1)
			}
			out.Text = append(out.Text, note)
		}
		if sheet != nil {
			var encoded bytes.Buffer
			for _, quality := range []int{82, 60, 40} {
				encoded.Reset()
				if err := jpeg.Encode(&encoded, sheet, &jpeg.Options{Quality: quality}); err != nil {
					return RecordEvidence{}, err
				}
				if encoded.Len() <= 512<<10 {
					break
				}
			}
			if encoded.Len() > 512<<10 {
				return RecordEvidence{}, fmt.Errorf("image sheet exceeds 512 KiB")
			}
			out.Images = append(out.Images, llm.ImagePart{Mime: "image/jpeg", Base64: base64.StdEncoding.EncodeToString(encoded.Bytes())})
		}
	}
	return out, nil
}

func drawTile(sheet *image.RGBA, raw []byte, origin image.Point, cell int) error {
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > (16<<20)/config.Height {
		return fmt.Errorf("tile source exceeds pixel budget")
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	longest := max(config.Width, config.Height)
	w, h := max(1, config.Width*cell/longest), max(1, config.Height*cell/longest)
	pos := origin.Add(image.Pt((cell-w)/2, (cell-h)/2))
	draw.ApproxBiLinear.Scale(sheet, image.Rectangle{Min: pos, Max: pos.Add(image.Pt(w, h))}, src, src.Bounds(), draw.Over, nil)
	return nil
}
