//go:build fts5

package convert

import (
	"fmt"
	"io"
	"net/mail"
	"os"
)

// EmailParser extracts text from .eml/.msg email files.
type EmailParser struct{}

func (p *EmailParser) Extensions() []string { return []string{".eml", ".msg"} }

func (p *EmailParser) Extract(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("extract email: %w", err)
	}
	defer f.Close()
	msg, err := mail.ReadMessage(f)
	if err != nil {
		return "", fmt.Errorf("extract email: %w", err)
	}
	var text string
	text += "From: " + msg.Header.Get("From") + "\n"
	text += "To: " + msg.Header.Get("To") + "\n"
	text += "Subject: " + msg.Header.Get("Subject") + "\n"
	text += "Date: " + msg.Header.Get("Date") + "\n\n"
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return "", fmt.Errorf("extract email: read body: %w", err)
	}
	text += string(body)
	return text, nil
}
