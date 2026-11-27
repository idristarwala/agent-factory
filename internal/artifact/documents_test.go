package artifact

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPDFPages(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("standalone binary requires poppler-utils for PDF")
	}
	for _, name := range []string{"pages", "blank"} {
		data, err := os.ReadFile("testdata/" + name + ".pdf")
		if err != nil {
			t.Fatal(err)
		}
		passages, err := pdfPassages(data)
		if name == "blank" {
			if err == nil || !strings.Contains(err.Error(), "no extractable text") {
				t.Fatal(err)
			}
			continue
		}
		if err != nil || len(passages) != 1 || passages[0].Location != "page 2" || passages[0].Text != "The expected hue is blue." {
			t.Fatal(passages, err)
		}
	}
}
func TestWordParagraphs(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	body, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = body.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p/><w:p><w:r><w:t>preserve</w:t><w:tab/><w:t>order</w:t></w:r></w:p></w:body></w:document>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	passages, err := wordPassages(buffer.Bytes())
	if err != nil || len(passages) != 1 || passages[0].Location != "paragraph 2" || passages[0].Text != "preserve order" {
		t.Fatal(passages, err)
	}
}
