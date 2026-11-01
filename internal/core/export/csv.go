package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StreamCSV writes records as a downloadable CSV with UTF-8 BOM for Excel compatibility
func StreamCSV(c *gin.Context, filename string, headers []string, rows [][]string) {
	buf := new(bytes.Buffer)
	// UTF-8 BOM so Excel properly opens Indonesian accents and UTF-8 symbols
	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(buf)
	if err := writer.Write(headers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV header"})
		return
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV row"})
			return
		}
	}
	writer.Flush()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}
