// Package export writes posts to JSON/CSV. It is shared by the postsvc
// business logic and the gateway (which formats posts fetched over RPC),
// so the two never diverge in output format.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"Post_Analyzer_Webserver/internal/models"
)

func WriteJSON(w io.Writer, posts []models.Post) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(posts)
}

func WriteCSV(w io.Writer, posts []models.Post) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	if err := csvWriter.Write([]string{"ID", "UserID", "Title", "Body", "CreatedAt", "UpdatedAt"}); err != nil {
		return err
	}

	for _, post := range posts {
		row := []string{
			fmt.Sprintf("%d", post.ID),
			fmt.Sprintf("%d", post.UserID),
			post.Title,
			post.Body,
			post.CreatedAt.Format(time.RFC3339),
			post.UpdatedAt.Format(time.RFC3339),
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func Write(w io.Writer, format models.ExportFormat, posts []models.Post) error {
	switch format {
	case models.ExportFormatJSON:
		return WriteJSON(w, posts)
	case models.ExportFormatCSV:
		return WriteCSV(w, posts)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}
