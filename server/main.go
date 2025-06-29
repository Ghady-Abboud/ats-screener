package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/gin-gonic/gin"
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
	resume, err := extractResumeData(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"status": "success",
		"file":   file.Filename,
		"data":   resume,
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

func extractResumeData(content string) (*Resume, error) {
	resume := &Resume{}
	emailRegex := `[a-zA-Z0-9-_]{1,}@[a-zA-Z0-9-_]{1,}.[a-zA-Z]{1,}`
	phoneRegex := `\d{3}-\d{3}-\d{4}`

	if phoneMatch := regexp.MustCompile(phoneRegex).FindString(content); phoneMatch != "" {
		resume.PhoneNumber = phoneMatch
	}
	if emailMatch := regexp.MustCompile(emailRegex).FindAllString(content, -1); len(emailMatch) > 0 {
		resume.Email = emailMatch[0]
	}
	education := extractEducationData(content)
	if len(education.SchoolName) > 0 {
		resume.Education = education
	}
	return resume, nil
}

func extractEducationData(content string) Education {
	education := Education{}
	schoolRegex := `([A-Z][^\s,.]+[.]?\s[(]?)*(College|University|Institute|Law School|School of|Academy)[^,\d]*[?=,|\d]`
	schoolMatches := regexp.MustCompile(schoolRegex).FindAllString(content, -1)
	for _, match := range schoolMatches {
		if len(match) > 2 && len(match) < 100 {
			education.SchoolName = append(education.SchoolName, match)
		}
	}
	gpa := extractGPA(content)
	if len(gpa) > 0 {
		education.GPA = gpa
	}
	return education
}

func extractGPA(content string) []string {
	var gpas []string
	seenGPAs := make(map[string]bool)

	gpaPatterns := []string{
		`(?i)gpa[\s:]*([0-4]\.[0-9]{1,2})\s*(?:/\s*4\.0)?`, 
        `(?i)gpa[\s:]*([0-4]\.[0-9]{1,2})\s*/\s*([0-4]\.[0-9]{1,2})`, 
        `(?i)cumulative\s+gpa[\s:]*([0-4]\.[0-9]{1,2})`,     
        `(?i)overall\s+gpa[\s:]*([0-4]\.[0-9]{1,2})`,                
	}
	for _, pattern := range gpaPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				gpa := strings.TrimSpace(match[1])
				if !seenGPAs[gpa] && isValidGPA(gpa) {
					gpas = append(gpas, gpa)
					seenGPAs[gpa] = true
				}
			}
		}
	}
	return gpas
}

func isValidGPA(gpa string) bool {
	if len(gpa) < 2 || len(gpa) > 4 {
		return false
	}
	return true
}
func main() {
	router := gin.Default()
	router.POST("/upload", uploadFile)

	router.Run("localhost:8080")
}
