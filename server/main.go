package main

import (
	"bytes"
	"encoding/json"
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
	SchoolName interface{} `json:"school_name,omitempty"`
	GPA        interface{} `json:"gpa,omitempty"`
	Degrees    interface{} `json:"degrees,omitempty"`
	Courses    interface{} `json:"courses,omitempty"`
}

type Resume struct {
	Email         string      `json:"email,omitempty"`
	PhoneNumber   string      `json:"phone_number,omitempty"`
	ExternalLinks interface{} `json:"external_links,omitempty"`
	Experience    interface{} `json:"experience,omitempty"`
	Projects      interface{} `json:"projects,omitempty"`
	Skills        interface{} `json:"skills,omitempty"`
	Interests     interface{} `json:"interests,omitempty"`
	Publications  interface{} `json:"publications,omitempty"`
	Education     interface{} `json:"education,omitempty"`
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
		return
	}

	systemPrompt := `Parse this resume and return ONLY a JSON object. Use arrays for: external_links, experience, projects, skills, interests, publications. For education use: school_name (string), gpa (string), degrees (array), courses (array). Return only JSON, no explanations. Resume: ` + content

	response, err := client.CreateChatCompletion(groq.CompletionCreateParams{
		Model:          "deepseek-r1-distill-llama-70b",
		ResponseFormat: groq.ResponseFormat{Type: "json_object"},
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
	systemResponse := response.Choices[0].Message.Content
	resume, err := JsonToResume(systemResponse)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"status": "success",
		"file":   file.Filename,
		"resume": resume,
	})
}

func JsonToResume(jsonData string) (*Resume, error) {
	var resume Resume
	err := json.Unmarshal([]byte(jsonData), &resume)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return nil, err
	}
	return &resume, nil

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
