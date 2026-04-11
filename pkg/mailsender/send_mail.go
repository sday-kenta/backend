package mailsender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/sday-kenta/backend/internal/entity"
)

// MaskEmail hides the local part for logs (e.g. j***@mail.ru).
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 1 || at == len(email)-1 {
		return "***"
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 1 {
		return "*" + domain
	}
	return string(local[0]) + "***" + domain
}

func SendMail(subject string, body string, to []string) error {
	smtpMailName, smtpMailCode, err := smtpCredentials()
	if err != nil {
		slog.Warn("mailsender.SendMail: smtp credentials", "err", err)
		return err
	}
	smtpHost, smtpAddr := smtpServerConfig()

	masked := make([]string, len(to))
	for i, addr := range to {
		masked[i] = MaskEmail(addr)
	}
	slog.Info("mailsender.SendMail: sending",
		"smtp_addr", smtpAddr,
		"smtp_host", smtpHost,
		"from_configured", smtpMailName != "",
		"to_count", len(to),
		"to_masked", strings.Join(masked, ","),
		"subject", subject,
	)

	auth := smtp.PlainAuth(
		"",
		smtpMailName,
		smtpMailCode,
		smtpHost,
	)

	msg := buildPlainTextMessage(subject, body, smtpMailName, to)

	if err = smtp.SendMail(
		smtpAddr,
		auth,
		smtpMailName,
		to,
		msg,
	); err != nil {
		slog.Error("mailsender.SendMail: smtp send failed",
			"smtp_addr", smtpAddr,
			"err", err,
		)
		return fmt.Errorf("send smtp mail: %w", err)
	}

	slog.Info("mailsender.SendMail: sent ok", "smtp_addr", smtpAddr, "to_count", len(to))
	return nil
}

// SendMailWithAttachment sends an HTML email with optional inline resources and attachments.
func SendMailWithAttachment(subject, htmlBody string, to []string, attachmentName string, attachment []byte, attachmentContentType string, inlineAttachments []entity.InlineAttachment) error {
	smtpMailName, smtpMailCode, err := smtpCredentials()
	if err != nil {
		slog.Warn("mailsender.SendMailWithAttachment: smtp credentials", "err", err)
		return err
	}
	smtpHost, smtpAddr := smtpServerConfig()

	masked := make([]string, len(to))
	for i, addr := range to {
		masked[i] = MaskEmail(addr)
	}
	slog.Info("mailsender.SendMailWithAttachment: sending",
		"smtp_addr", smtpAddr,
		"smtp_host", smtpHost,
		"from_configured", smtpMailName != "",
		"to_count", len(to),
		"to_masked", strings.Join(masked, ","),
		"subject", subject,
		"attachment_bytes", len(attachment),
		"inline_parts", len(inlineAttachments),
	)

	auth := smtp.PlainAuth("", smtpMailName, smtpMailCode, smtpHost)

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

	if err := smtp.SendMail(smtpAddr, auth, smtpMailName, to, message.Bytes()); err != nil {
		slog.Error("mailsender.SendMailWithAttachment: smtp send failed",
			"smtp_addr", smtpAddr,
			"err", err,
		)
		return err
	}
	slog.Info("mailsender.SendMailWithAttachment: sent ok", "smtp_addr", smtpAddr, "to_count", len(to))
	return nil
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

func smtpCredentials() (string, string, error) {
	smtpMailName := strings.TrimSpace(os.Getenv("SMTP_MAIL"))
	smtpMailCode := normalizeSMTPPassword(os.Getenv("SMTP_CODE"))
	if smtpMailName == "" || smtpMailCode == "" {
		return "", "", fmt.Errorf("smtp credentials are not configured")
	}

	return smtpMailName, smtpMailCode, nil
}

func smtpServerConfig() (string, string) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		host = "smtp.mail.ru"
	}

	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}

	return host, host + ":" + port
}

func normalizeSMTPPassword(raw string) string {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "")
}

func buildPlainTextMessage(subject, body, from string, to []string) []byte {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", subject)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}

	return []byte(strings.Join(headers, "\r\n"))
}
