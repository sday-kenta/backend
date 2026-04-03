package mailsender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/sday-kenta/backend/internal/entity"
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
		return err
	}

	return nil
}

// SendMailWithAttachment sends an HTML email with optional inline resources and attachments.
func SendMailWithAttachment(subject, htmlBody string, to []string, attachmentName string, attachment []byte, attachmentContentType string, inlineAttachments []entity.InlineAttachment) error {
	smtpMailName := os.Getenv("SMTP_MAIL")
	smtpMailCode := os.Getenv("SMTP_CODE")
	if smtpMailName == "" || smtpMailCode == "" {
		return fmt.Errorf("smtp credentials are not configured")
	}

	auth := smtp.PlainAuth("", smtpMailName, smtpMailCode, "smtp.gmail.com")

	var message bytes.Buffer
	writer := multipart.NewWriter(&message)
	relatedBoundary := ""

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

	if len(inlineAttachments) > 0 {
		var relatedMessage bytes.Buffer
		relatedWriter := multipart.NewWriter(&relatedMessage)
		relatedBoundary = relatedWriter.Boundary()

		htmlHeader := textproto.MIMEHeader{}
		htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
		htmlHeader.Set("Content-Transfer-Encoding", "8bit")
		htmlPart, err := relatedWriter.CreatePart(htmlHeader)
		if err != nil {
			return err
		}
		if _, err = htmlPart.Write([]byte(htmlBody)); err != nil {
			return err
		}
		if _, err = htmlPart.Write([]byte("\r\n")); err != nil {
			return err
		}

		for _, inlineAttachment := range inlineAttachments {
			inlineHeader := textproto.MIMEHeader{}
			inlineHeader.Set("Content-Type", inlineAttachment.ContentType)
			inlineHeader.Set("Content-Transfer-Encoding", "base64")
			inlineHeader.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(inlineAttachment.FileName)))
			inlineHeader.Set("Content-ID", fmt.Sprintf("<%s>", inlineAttachment.ContentID))

			inlinePart, createErr := relatedWriter.CreatePart(inlineHeader)
			if createErr != nil {
				return createErr
			}
			if err = writeBase64Body(inlinePart, inlineAttachment.Body); err != nil {
				return err
			}
		}

		if err = relatedWriter.Close(); err != nil {
			return err
		}

		relatedHeader := textproto.MIMEHeader{}
		relatedHeader.Set("Content-Type", fmt.Sprintf(`multipart/related; boundary=%s`, relatedBoundary))
		relatedPart, err := writer.CreatePart(relatedHeader)
		if err != nil {
			return err
		}
		if _, err = relatedPart.Write(relatedMessage.Bytes()); err != nil {
			return err
		}
	} else {
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
		if writeErr := writeBase64Body(attachmentPart, attachment); writeErr != nil {
			return writeErr
		}
	}

	if closeErr := writer.Close(); closeErr != nil {
		return closeErr
	}

	return smtp.SendMail("smtp.gmail.com:587", auth, smtpMailName, to, message.Bytes())
}

func writeBase64Body(dst io.Writer, content []byte) error {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
	base64.StdEncoding.Encode(encoded, content)
	for len(encoded) > 76 {
		if _, err := dst.Write(encoded[:76]); err != nil {
			return err
		}
		if _, err := dst.Write([]byte("\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := dst.Write(encoded); err != nil {
			return err
		}
		if _, err := dst.Write([]byte("\r\n")); err != nil {
			return err
		}
	}
	return nil
}
