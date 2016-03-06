package mail

import (
	"encoding/base64"
	"net/mail"
	"net/smtp"
	"strings"
)

type Mail struct {
	smtpServer string
	smtpPort   string
	from       string
	auth       smtp.Auth
}

func NewMail(smtpServer, port, acount, password string) *Mail {
	mail := new(Mail)
	mail.smtpServer = smtpServer
	mail.smtpPort = port
	mail.from = acount
	mail.auth = smtp.PlainAuth(
		"",
		acount,
		password,
		smtpServer,
	)

	return mail
}

func (mail *Mail) Send(to, title, msg string) error {
	content := "From: " + mail.from
	content += "\r\n"
	content += "To: " + to
	content += "\r\n"
	content += "Subject: " + encodeRFC2047(title)
	content += "\r\n"
	content += "MIME-Version: 1.0"
	content += "\r\n"
	content += "Content-Type: text/plain"
	content += "\r\n"
	content += "Content-Transfer-Encoding: base64"
	content += "\r\n"
	content += "\r\n"
	content += base64.StdEncoding.EncodeToString([]byte(msg))

	err := smtp.SendMail(
		mail.smtpServer+":"+mail.smtpPort,
		mail.auth,
		mail.from,
		strings.Split(to, ","),
		[]byte(content),
	)

	if err != nil {
		return err
	}

	return nil
}

func encodeRFC2047(str string) string {
	addr := mail.Address{str, ""}
	return strings.Trim(addr.String(), " <>")
}
