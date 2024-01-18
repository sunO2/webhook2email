package main

import (
	"fmt"
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
	sendTos := []string{smtpSendTo}
	if len(sendTo) > 0 {
		sendTos = strings.Split(sendTo, ",")
	}

	msg := fmt.Sprintf("From: %s\r\nSubject: %s\r\nTo: %s\r\nContent-Type:text/html;charset=UTF-8\r\n\r\n%s",
		name,
		title,
		sendTo,
		message)

	auth := smtp.PlainAuth("", from, password, host)
	err := smtp.SendMail(fmt.Sprintf("%s:%s", host, port), auth, from,
		sendTos, []byte(msg))
	if nil != err {
		log.Println("--------》》》》》", err)
		rw.WriteHeader(501)
		rw.Write([]byte(err.Error()))
	} else {
		log.Println("发送成功 >>> [", sendTo, "]")
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
	http.ListenAndServe(":80", nil)
}
