package artifact

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const documentTextLimit = 32 << 10

type passage struct {
	Location string `json:"location"`
	Text     string `json:"text"`
}

func pdfPassages(data []byte) ([]passage, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, fmt.Errorf("invalid PDF header")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdftotext", "-enc", "UTF-8", "-", "-")
	cmd.Stdin = bytes.NewReader(data)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); errors.Is(err, exec.ErrNotFound) {
		return nil, fmt.Errorf("PDF reading requires pdftotext on the API server")
	} else if err != nil {
		return nil, err
	}
	text, readErr := io.ReadAll(io.LimitReader(pipe, documentTextLimit+1))
	if len(text) > documentTextLimit {
		cancel()
	}
	err = cmd.Wait()
	if len(text) > documentTextLimit {
		return nil, fmt.Errorf("PDF text exceeds 32 KiB; attach a shorter excerpt")
	}
	if readErr != nil || err != nil {
		return nil, fmt.Errorf("could not read PDF text; check that the file opens without a password")
	}
	pages := strings.Split(strings.TrimSuffix(string(text), "\f"), "\f")
	var out []passage
	for i, value := range pages {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, passage{fmt.Sprintf("page %d", i+1), value})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("PDF has no extractable text; attach an OCR version or a screenshot")
	}
	return out, nil
}

func wordPassages(data []byte) ([]passage, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid DOCX archive")
	}
	body, err := archive.Open("word/document.xml")
	if err != nil {
		return nil, fmt.Errorf("DOCX document body is missing")
	}
	defer body.Close()
	const maxXML = 2 << 20
	raw, err := io.ReadAll(io.LimitReader(body, maxXML+1))
	if err != nil || len(raw) > maxXML {
		return nil, fmt.Errorf("DOCX document body exceeds limits or is unreadable")
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var out []passage
	var paragraph strings.Builder
	var number, total int
	inParagraph := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid DOCX document XML")
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if !wordElement(elem.Name) {
				continue
			}
			switch elem.Name.Local {
			case "p":
				number++
				inParagraph = true
				paragraph.Reset()
			case "t":
				if inParagraph {
					var value string
					if err := decoder.DecodeElement(&value, &elem); err != nil {
						return nil, fmt.Errorf("invalid DOCX text")
					}
					paragraph.WriteString(value)
				}
			case "br", "tab":
				if inParagraph {
					paragraph.WriteByte(' ')
				}
			}
		case xml.EndElement:
			if inParagraph && wordElement(elem.Name) && elem.Name.Local == "p" {
				inParagraph = false
				text := strings.TrimSpace(paragraph.String())
				if text != "" {
					location := fmt.Sprintf("paragraph %d", number)
					total += len(text) + len(location)
					if total > documentTextLimit {
						return nil, fmt.Errorf("DOCX text exceeds 32 KiB; attach a shorter excerpt")
					}
					out = append(out, passage{location, text})
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("DOCX has no readable body text")
	}
	return out, nil
}

func wordElement(name xml.Name) bool {
	return name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" || name.Space == "http://purl.oclc.org/ooxml/wordprocessingml/main"
}
