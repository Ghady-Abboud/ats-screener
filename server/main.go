package main

import (
	"bytes"
	"mime/multipart"
	"net/http"

	"github.com/dslipak/pdf"
	"github.com/gin-gonic/gin"
)

func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := readFileContent(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is empty or not a valid PDF"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusText(http.StatusOK), "file": file.Filename})
}

func readFileContent(fileHeader *multipart.FileHeader) (string, error){
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	reader, err := pdf.NewReader(file, fileHeader.Size)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	b, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)
	return buf.String(), nil
}

func main() {
	router := gin.Default()
	router.POST("/upload", uploadFile)

	router.Run("localhost:8080")
}