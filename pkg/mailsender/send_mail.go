package mailsender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
)

func SendMail(subject string, body string, to []string) error {
	smtpMailName := os.Getenv("SMTP_MAIL")
	smtpMailCode := os.Getenv("SMTP_CODE")

	auth := smtp.PlainAuth(
		"",
		smtpMailName,
		smtpMailCode,
		"smtp.gmail.com",
	)

	msg := "Subject: " + subject + "\n" + body

	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		smtpMailName,
		to,
		[]byte(msg),
	)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	return nil
}

// SendMailWithAttachment sends an HTML email and optionally attaches a file.
func SendMailWithAttachment(subject, htmlBody string, to []string, attachmentName string, attachment []byte, attachmentContentType string) error {
	smtpMailName := os.Getenv("SMTP_MAIL")
	smtpMailCode := os.Getenv("SMTP_CODE")
	if smtpMailName == "" || smtpMailCode == "" {
		return fmt.Errorf("smtp credentials are not configured")
	}

	auth := smtp.PlainAuth("", smtpMailName, smtpMailCode, "smtp.gmail.com")

	var message bytes.Buffer
	writer := multipart.NewWriter(&message)

	headers := []string{
		fmt.Sprintf("From: %s", smtpMailName),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", subject)),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s", writer.Boundary()),
		"",
	}
	for _, header := range headers {
		_, _ = message.WriteString(header + "\r\n")
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlHeader.Set("Content-Transfer-Encoding", "8bit")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return err
	}
	if _, err = htmlPart.Write([]byte(htmlBody)); err != nil {
		return err
	}
	if _, err = htmlPart.Write([]byte("\r\n")); err != nil {
		return err
	}

	if len(attachment) > 0 {
		attachmentHeader := textproto.MIMEHeader{}
		attachmentHeader.Set("Content-Type", attachmentContentType)
		attachmentHeader.Set("Content-Transfer-Encoding", "base64")
		attachmentHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(attachmentName)))

		attachmentPart, createErr := writer.CreatePart(attachmentHeader)
		if createErr != nil {
			return createErr
		}

		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
		base64.StdEncoding.Encode(encoded, attachment)
		for len(encoded) > 76 {
			if _, err = attachmentPart.Write(encoded[:76]); err != nil {
				return err
			}
			if _, err = attachmentPart.Write([]byte("\r\n")); err != nil {
				return err
			}
			encoded = encoded[76:]
		}
		if len(encoded) > 0 {
			if _, err = attachmentPart.Write(encoded); err != nil {
				return err
			}
			if _, err = attachmentPart.Write([]byte("\r\n")); err != nil {
				return err
			}
		}
	}

	if err = writer.Close(); err != nil {
		return err
	}

	return smtp.SendMail("smtp.gmail.com:587", auth, smtpMailName, to, message.Bytes())
}
