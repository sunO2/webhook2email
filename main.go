package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

var (
	from       string
	host       string
	name       string
	port       string
	password   string
	smtpSendTo string
)

func webHookToEmailHandler(rw http.ResponseWriter, request *http.Request) {
	title := request.URL.Query().Get("title")
	message := request.URL.Query().Get("message")
	sendTo := request.URL.Query().Get("sendTo")
	actionUrl := request.URL.Query().Get("action-url")

	sendTos := []string{smtpSendTo}
	if len(sendTo) > 0 {
		sendTos = strings.Split(sendTo, ",")
	}
	tmpl, _ := template.ParseFiles("./mail.html")
	htmlTemplate := &strings.Builder{}

	data := struct {
		Title     string
		From      string
		Message   template.HTML
		ActionUrl string
	}{
		Title:     title,
		Message:   template.HTML(message),
		From:      from,
		ActionUrl: actionUrl,
	}

	tmpl.Execute(htmlTemplate, data)
	message = htmlTemplate.String()

	var emailTemplate = "From:" + name + "\n" +
		"Subject:" + title + "\n" +
		"To: " + sendTo + "\n" +
		"Content-Type:text/html; charset=UTF-8;" + "\n\n" + message

	auth := smtp.PlainAuth("", from, password, host)
	err := smtp.SendMail(fmt.Sprintf("%s:%s", host, port), auth, from,
		sendTos, []byte(emailTemplate))
	if nil != err {
		log.Println("--------》》》》》", err)
		rw.WriteHeader(501)
		rw.Write([]byte(err.Error()))
	} else {
		log.Println("发送成功 >>>", sendTos)
		rw.WriteHeader(200)
		rw.Write([]byte("发送成功"))
	}
}

func main() {
	host = os.Getenv("SMTP_HOST")
	port = os.Getenv("SMTP_PORT")
	password = os.Getenv("SMTP_PASSWORD")
	from = os.Getenv("SMTP_FROM")
	smtpSendTo = os.Getenv("SMTP_SEND_TO")
	name = os.Getenv("SMTP_FROM_NAME")

	http.HandleFunc("/webhook2email", webHookToEmailHandler)
	err := http.ListenAndServe(":80", nil)
	if nil != err {
		fmt.Println(err)
	}
}
