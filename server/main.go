package main

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/dslipak/pdf"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/jpoz/groq"
)

type Education struct {
	SchoolName []string `json:"school_name,omitempty"`
	GPA        []string `json:"gpa,omitempty"`
	Degrees    []string `json:"degrees,omitempty"`
	Courses    []string `json:"courses,omitempty"`
}

type Resume struct {
	Email         string    `json:"email,omitempty"`
	PhoneNumber   string    `json:"phone_number,omitempty"`
	ExternalLinks []string  `json:"external_links,omitempty"`
	Experience    []string  `json:"experience,omitempty"`
	Projects      []string  `json:"projects,omitempty"`
	Skills        []string  `json:"skills,omitempty"`
	Interests     []string  `json:"interests,omitempty"`
	Education     Education `json:"education,omitempty"`
}

func GroqMiddleware(client *groq.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("groqClient", client)
		c.Next()
	}
}

func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := readPDFContent(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is empty or not a valid PDF"})
		return
	}

	clientInterface, exists := c.Get("groqClient")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Groq client not found"})
		return
	}

	client, ok := clientInterface.(*groq.Client)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid Groq Client"})
	}

	systemPrompt := "You are a resume parser. You take text as input and return a JSON object with the following fields: email, phone_number, external_links, experience, projects, skills, interests, education. The education field should be an object with school_name, gpa, degrees, and courses as arrays of strings. The input text is: " + content

	response, err := client.CreateChatCompletion(groq.CompletionCreateParams{
		Model:          "deepseek-r1-distill-llama-70b",
		ResponseFormat: groq.ResponseFormat{Type: "text"},
		Messages: []groq.Message{
			{
				Role:    "user",
				Content: systemPrompt,
			},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("Response: ", response.Choices[0].Message.Content)
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": "success",
		"file":   file.Filename,
		"data":   content,
	})
}

func readPDFContent(fileHeader *multipart.FileHeader) (string, error) {
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
	godotenv.Load()
	router := gin.Default()
	client := groq.NewClient(groq.WithAPIKey(os.Getenv("GROQ_API_KEY")))

	router.Use(GroqMiddleware(client))
	router.POST("/upload", uploadFile)

	router.Run("localhost:8080")
}
