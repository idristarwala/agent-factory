package llm

import "encoding/base64"

// ImagePart represents an image for multimodal models.
// Adapters can include these in the request when the gateway supports vision.
type ImagePart struct {
	URL    string
	Base64 string
	Mime   string
}

// WithImages is a helper that adapters can use to attach images to a request.
// Full wiring into OpenAI/Gemini/Anthropic request bodies is an incremental step.
func WithImages(parts []ImagePart) []ImagePart {
	return parts
}

// imageBlocks encodes inline evidence for a provider without delegating URL fetching to it.
func (m Message) imageBlocks(protocol string) ([]map[string]any, error) {
	var blocks []map[string]any
	for _, image := range m.Images {
		if m.Role != "user" || image.Base64 == "" || image.URL != "" {
			return nil, &Error{Type: "BAD_REQUEST", Message: "image evidence requires inline data on a user turn"}
		}
		switch image.Mime {
		case "image/png", "image/jpeg", "image/webp":
		default:
			return nil, &Error{Type: "BAD_REQUEST", Message: "unsupported image MIME type"}
		}
		if _, err := base64.StdEncoding.DecodeString(image.Base64); err != nil {
			return nil, &Error{Type: "BAD_REQUEST", Message: "invalid base64 image evidence"}
		}
		var block map[string]any
		switch protocol {
		case "openai":
			block = map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + image.Mime + ";base64," + image.Base64}}
		case "gemini":
			block = map[string]any{"inlineData": map[string]string{"mimeType": image.Mime, "data": image.Base64}}
		case "anthropic":
			block = map[string]any{"type": "image", "source": map[string]string{"type": "base64", "media_type": image.Mime, "data": image.Base64}}
		default:
			return nil, &Error{Type: "BAD_REQUEST", Message: "unsupported image protocol"}
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// UserParts returns text followed by inline images in the selected protocol.
func (m Message) UserParts(protocol string) ([]map[string]any, error) {
	images, err := m.imageBlocks(protocol)
	if err != nil {
		return nil, err
	}
	if m.Content == "" {
		return images, nil
	}
	text := map[string]any{"text": m.Content}
	if protocol != "gemini" {
		text["type"] = "text"
	}
	return append([]map[string]any{text}, images...), nil
}
