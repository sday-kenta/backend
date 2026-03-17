package mailsender

import (
	"fmt"
	"net/smtp"
	"os"
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
		fmt.Println(err)
	}

	return nil
}
