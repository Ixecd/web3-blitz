package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// Mailer 通过环境变量配置 SMTP。
//
//	SMTP_HOST   smtp 服务器地址（默认 smtp.gmail.com）
//	SMTP_PORT   端口：465=隐式 TLS，587=STARTTLS（默认 465）
//	SMTP_USER   认证用户名
//	SMTP_PASS   认证密码 / 授权码
//	SMTP_FROM   发件人地址
//	FRONTEND_URL 前端地址，用于拼接重置链接（默认 http://localhost:5173）
type Mailer struct {
	host        string
	port        string
	username    string
	password    string
	from        string
	frontendURL string
}

func NewMailer() *Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "465"
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	return &Mailer{
		host:        host,
		port:        port,
		username:    os.Getenv("SMTP_USER"),
		password:    os.Getenv("SMTP_PASS"),
		from:        os.Getenv("SMTP_FROM"),
		frontendURL: frontendURL,
	}
}

// SendResetEmail 发送密码重置邮件，链接指向前端 /reset-password?token=<token>。
func (m *Mailer) SendResetEmail(to, token string) error {
	resetURL := m.frontendURL + "/reset-password?token=" + token
	subject := "BLITZ - 密码重置"
	body := fmt.Sprintf(
		"您好，\r\n\r\n"+
			"您已请求重置密码。请点击以下链接完成重置（链接 30 分钟内有效）：\r\n\r\n"+
			"%s\r\n\r\n"+
			"如果您没有请求重置密码，请忽略此邮件。\r\n\r\n"+
			"— BLITZ 团队",
		resetURL,
	)
	return m.send(to, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := net.JoinHostPort(m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	// port 465：隐式 TLS
	if m.port == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, m.host)
		if err != nil {
			return fmt.Errorf("smtp client: %w", err)
		}
		defer client.Close()
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err := client.Mail(m.from); err != nil {
			return fmt.Errorf("smtp mail from: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		defer w.Close()
		_, err = fmt.Fprint(w, msg)
		return err
	}

	// port 587：STARTTLS
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
