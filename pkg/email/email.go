package email

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
)

//go:embed templates/*.html
var templateFS embed.FS

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
	appURL       string
}

type PasswordResetData struct {
	ResetLink   string
	UserEmail   string
	ExpiryHours int
}

func NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, fromEmail, fromName, appURL string) *EmailService {
	return &EmailService{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		fromName:     fromName,
		appURL:       appURL,
	}
}

func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
	// Create reset link
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, resetToken)

	// Prepare template data
	data := PasswordResetData{
		ResetLink:   resetLink,
		UserEmail:   to,
		ExpiryHours: 24,
	}

	// Parse template
	tmpl, err := template.ParseFS(templateFS, "templates/password_reset.html")
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Execute template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Prepare email message
	subject := "Password Reset Request - Fleet Backend"
	message := s.buildEmailMessage(to, subject, body.String())

	// Send email
	if err := s.sendEmail(to, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *EmailService) buildEmailMessage(to, subject, htmlBody string) []byte {
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	return []byte(message)
}

func (s *EmailService) sendEmail(to string, message []byte) error {
	// Set up authentication
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	// TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.smtpHost,
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	// For port 587 (TLS), use STARTTLS
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Start TLS
	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Set sender
	if err = conn.Mail(s.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err = conn.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return conn.Quit()
}
