package gmail

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
)

type Gmail struct {
	from string
	auth smtp.Auth
}

func NewGmail(acount, password string) *Gmail {
	gm := new(Gmail)
	gm.from = acount
	gm.auth = smtp.PlainAuth(
		"",
		acount,
		password,
		"smtp.gmail.com",
	)

	return gm
}

func (gm *Gmail) Send(to, title, msg string) {

	content := "From: " + gm.from
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
		"smtp.gmail.com:587",
		gm.auth,
		gm.from,
		[]string{to},
		[]byte(content),
	)

	if err != nil {
		fmt.Println(err)
	}
}

func encodeRFC2047(str string) string {
	addr := mail.Address{str, ""}
	return strings.Trim(addr.String(), " <>")
}
