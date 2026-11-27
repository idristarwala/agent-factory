package llm

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
