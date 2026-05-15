package utils

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"publimd/config"

	"github.com/resend/resend-go/v3"
)

func SendEmailSync(ctx context.Context, to []string, subject string, htmlBody string) error {
	log.Println("Intentando enviar correo vía Google SMTP...")
	err := sendWithGoogle(to, subject, htmlBody)
	if err == nil {
		return nil
	}

	log.Printf("Google SMTP falló: %v. Reintentando con Resend...", err)

	err = sendWithResend(ctx, to, subject, htmlBody)
	if err != nil {
		log.Printf("Resend también falló: %v", err)
		return fmt.Errorf("todos los servicios de email fallaron: %w", err)
	}

	log.Println("Correo enviado exitosamente vía Resend")
	return nil
}

func sendWithGoogle(to []string, subject string, htmlBody string) error {
	emailConfig := config.GetEmailConfig()

	msg := fmt.Appendf(nil, "To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"+
		"%s",
		to[0], emailConfig.From, subject, htmlBody)

	auth := smtp.PlainAuth("", emailConfig.SMTPUser, emailConfig.SMTPPass, emailConfig.SMTPHost)
	addr := fmt.Sprintf("%s:%s", emailConfig.SMTPHost, emailConfig.SMTPPort)

	return smtp.SendMail(addr, auth, emailConfig.From, to, msg)
}

func sendWithResend(ctx context.Context, to []string, subject string, htmlBody string) error {
	emailConfig := config.GetEmailConfig()
	client := resend.NewClient(emailConfig.SMTPPassKey)

	params := &resend.SendEmailRequest{
		From:    "No Reply support@mmabitec.me",
		To:      to,
		Html:    htmlBody,
		Subject: subject,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
