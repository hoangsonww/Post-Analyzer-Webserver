package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"Post_Analyzer_Webserver/internal/models"
)

func samplePosts() []models.Post {
	t := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	return []models.Post{
		{ID: 1, UserID: 10, Title: "First", Body: "Body one", CreatedAt: t, UpdatedAt: t},
		{ID: 2, UserID: 11, Title: "Second, with comma", Body: "Body \"two\"", CreatedAt: t, UpdatedAt: t},
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, samplePosts()); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var got []models.Post
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output isn't valid JSON: %v", err)
	}
	if len(got) != 2 || got[0].Title != "First" || got[1].Title != "Second, with comma" {
		t.Fatalf("unexpected JSON round trip: %+v", got)
	}
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, samplePosts()); err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("output isn't valid CSV: %v", err)
	}
	if len(records) != 3 { // header + 2 rows
		t.Fatalf("expected 3 CSV records (header+2), got %d: %v", len(records), records)
	}
	if records[0][0] != "ID" {
		t.Errorf("expected header row to start with ID, got %v", records[0])
	}
	// Row with an embedded comma/quote must survive the CSV round trip
	// intact — this is exactly the case encoding/csv exists to handle.
	if records[2][2] != "Second, with comma" {
		t.Errorf("comma-containing title didn't round-trip: got %q", records[2][2])
	}
}

func TestWrite_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, models.ExportFormat("xml"), samplePosts())
	if err == nil {
		t.Fatal("expected an error for an unsupported export format")
	}
}

func TestWrite_EmptyPosts(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, models.ExportFormatJSON, nil); err != nil {
		t.Fatalf("expected empty post list to export cleanly, got: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "[]" && strings.TrimSpace(buf.String()) != "null" {
		t.Errorf("expected an empty JSON array for zero posts, got %q", buf.String())
	}
}
