package export

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Service struct {
	pandocPath string
}

func NewService() *Service {
	path, _ := exec.LookPath("pandoc")
	return &Service{
		pandocPath: path,
	}
}

func (s *Service) Export(mdContent string, format string) ([]byte, string, error) {
	if s.pandocPath == "" {
		return nil, "", fmt.Errorf("pandoc not found on system")
	}

	var outputFormat string
	var contentType string

	switch format {
	case "pdf":
		outputFormat = "pdf"
		contentType = "application/pdf"
	case "docx":
		outputFormat = "docx"
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "html":
		outputFormat = "html"
		contentType = "text/html"
	case "markdown", "md":
		return []byte(mdContent), "text/markdown", nil
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	cmd := exec.Command(s.pandocPath, "--from", "markdown", "--to", outputFormat, "-o", "-")
	cmd.Stdin = bytes.NewBufferString(mdContent)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, "", fmt.Errorf("pandoc error: %v, stderr: %s", err, stderr.String())
	}

	return out.Bytes(), contentType, nil
}
