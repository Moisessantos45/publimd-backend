package utils

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/yuin/goldmark"
)

func GeneratePDFFromMarkdownContent(md string, outputPath string) error {
	var html bytes.Buffer

	if err := goldmark.Convert([]byte(md), &html); err != nil {
		return err
	}

	tmpHTML, err := os.CreateTemp("", "md-*.html")
	if err != nil {
		return err
	}
	defer os.Remove(tmpHTML.Name())

	if _, err := tmpHTML.Write(html.Bytes()); err != nil {
		tmpHTML.Close()
		return err
	}
	if err := tmpHTML.Close(); err != nil {
		return err
	}

	cmd := exec.Command("wkhtmltopdf", tmpHTML.Name(), outputPath)
	return cmd.Run()
}
