package mailsender

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

func smtpConnectTimeout() time.Duration {
	s := strings.TrimSpace(os.Getenv("SMTP_CONNECT_TIMEOUT"))
	if s == "" {
		return 20 * time.Second
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 20 * time.Second
	}
	return d
}

func smtpTotalTimeout() time.Duration {
	s := strings.TrimSpace(os.Getenv("SMTP_TOTAL_TIMEOUT"))
	if s == "" {
		return 90 * time.Second
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 90 * time.Second
	}
	return d
}

func validateSMTPLine(line string) error {
	if strings.ContainsAny(line, "\n\r") {
		return errors.New("smtp: a line must not contain CR or LF")
	}
	return nil
}

func smtpImplicitTLS(smtpAddr string) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("SMTP_IMPLICIT_TLS")), "true") {
		return true
	}
	_, port, err := net.SplitHostPort(smtpAddr)
	if err != nil {
		return false
	}
	return port == "465"
}

func dialSMTPTransport(smtpAddr, smtpHost string, connectTO time.Duration) (net.Conn, bool, error) {
	dialer := net.Dialer{Timeout: connectTO}
	if smtpImplicitTLS(smtpAddr) {
		conn, err := tls.DialWithDialer(&dialer, "tcp", smtpAddr, &tls.Config{ServerName: smtpHost})
		return conn, true, err
	}
	conn, err := dialer.Dial("tcp", smtpAddr)
	return conn, false, err
}

func formatDialErr(smtpAddr string, err error) error {
	if err == nil {
		return nil
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return fmt.Errorf("smtp dial %s: %w (timeout: allow outbound TCP from pod/VPC to this host:port, or use an HTTPS mail API relay)", smtpAddr, err)
	}
	return fmt.Errorf("smtp dial %s: %w", smtpAddr, err)
}

// sendMailSMTP mirrors net/smtp.SendMail but uses dial + session deadlines (stdlib Dial has no timeout).
// Port 465 (or SMTP_IMPLICIT_TLS=true) uses implicit TLS (SMTPS); other ports use plain TCP then STARTTLS when offered.
func sendMailSMTP(smtpAddr, smtpHost string, auth smtp.Auth, from string, to []string, msg []byte) error {
	if err := validateSMTPLine(from); err != nil {
		return err
	}
	for _, recp := range to {
		if err := validateSMTPLine(recp); err != nil {
			return err
		}
	}

	connectTO := smtpConnectTimeout()
	totalTO := smtpTotalTimeout()
	t0 := time.Now()

	conn, usedImplicitTLS, err := dialSMTPTransport(smtpAddr, smtpHost, connectTO)
	if err != nil {
		return formatDialErr(smtpAddr, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(totalTO)
	if err = conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("smtp set deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if usedImplicitTLS {
		slogSMTPPhase("smtp.smtps_ok", smtpAddr, time.Since(t0))
	} else {
		slogSMTPPhase("smtp.tcp_ok", smtpAddr, time.Since(t0))
	}

	if !usedImplicitTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: smtpHost}
			if err = client.StartTLS(config); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
			slogSMTPPhase("smtp.starttls_ok", smtpAddr, time.Since(t0))
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		slogSMTPPhase("smtp.auth_ok", smtpAddr, time.Since(t0))
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	slogSMTPPhase("smtp.payload_ok", smtpAddr, time.Since(t0))

	if err = client.Quit(); err != nil {
		return err
	}
	return nil
}

func slogSMTPPhase(event, smtpAddr string, elapsed time.Duration) {
	slog.Info("mailsender SMTP step",
		slog.String("component", "mailsender"),
		slog.String("event", event),
		slog.String("smtp_addr", smtpAddr),
		slog.Float64("elapsed_ms", float64(elapsed.Microseconds())/1000),
	)
}
