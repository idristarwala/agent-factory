package mapping

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDeclaredOpenAPIResponse(t *testing.T) {
	const raw = `openapi: 3.0.3
paths:
  /records:
    get:
      responses:
        "200":
          $ref: "#/components/responses/Page"
components:
  responses:
    Page:
      content:
        application/json:
          schema:
            type: object
            properties:
              rows:
                type: array
                items:
                  $ref: "#/components/schemas/Row"
  schemas:
    Row:
      type: object
      required: [id]
      properties:
        id: {type: integer}
        alias: {type: string, nullable: true, example: PRIVATE-VALUE}
`
	req := APIResponse{Document: raw, Path: "/records"}
	fields, err := req.Fields("/rows")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(fields)
	if len(fields) != 2 || fields[1].Path != "/id" || !fields[1].Required || strings.Contains(string(b), "PRIVATE-VALUE") || strings.Join(fields[0].Types, ",") != "null,string" {
		t.Fatal(string(b))
	}
	req.Document = strings.Replace(raw, "#/components/schemas/Row", "https://example.org/Row", 1)
	if _, err := req.Fields("/rows"); err == nil {
		t.Fatal("external ref accepted")
	}
	req.Document = strings.Replace(raw, "$ref: \"#/components/schemas/Row\"", "oneOf: []", 1)
	if _, err := req.Fields("/rows"); err == nil {
		t.Fatal("unsupported composition ignored")
	}
}
