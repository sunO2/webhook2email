package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"webhook2mail/mailbox"
)

var (
	from,
	name,
	host,
	port,
	imapHost,
	imapPort,
	password,
	smtpSendTo,
	defaultActionURL string
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

	message = createTemplateMessage(title, message, actionUrl)
	err := sendToEmail(title, message, sendTos)
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

func createTemplateMessage(title, message, actionUrl string) string {
	if len(defaultActionURL) > 0 && len(actionUrl) <= 0 {
		actionUrl = defaultActionURL
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
	return htmlTemplate.String()
}

func sendToEmail(title, message string, sendTos []string) error {
	sendTo := strings.Join(sendTos, ",")
	var emailTemplate = "From:" + name + "\n" +
		"Subject:" + title + "\n" +
		"To: " + sendTo + "\n" +
		"Content-Type:text/html; charset=UTF-8;" + "\n\n" + message

	auth := smtp.PlainAuth("", from, password, host)
	err := smtp.SendMail(fmt.Sprintf("%s:%s", host, port), auth, from,
		sendTos, []byte(emailTemplate))
	return err
}

// / 监听邮箱变化 并且转发
func createMailBox() {
	fmt.Println("开启IMAP Client")
	if client, err := mailbox.NewClient(imapHost, imapPort); nil != err {
		fmt.Println("客户端创建失败")
	} else {
		fmt.Println("开启IMAP Client成功")
		if err := client.Login(from, password); nil != err {
			fmt.Println("登陆失败")
		} else {
			go client.Idle(newMailMessageEvent)
		}
	}
}

func newMailMessageEvent(title, message, actionUrl string) {
	sendTos := []string{smtpSendTo}
	var templateMessage = createTemplateMessage(title, message, actionUrl)
	if err := sendToEmail(title, templateMessage, sendTos); nil != err {
		fmt.Println("转发失败")
	} else {
		fmt.Println("转发成功")
	}
}

func main() {
	host = os.Getenv("SMTP_HOST")
	port = os.Getenv("SMTP_PORT")

	imapHost = os.Getenv("IMAP_HOST")
	imapPort = os.Getenv("IMAP_PORT")

	password = os.Getenv("SMTP_PASSWORD")
	from = os.Getenv("SMTP_FROM")
	smtpSendTo = os.Getenv("SMTP_SEND_TO")
	name = os.Getenv("SMTP_FROM_NAME")
	defaultActionURL = os.Getenv("DEFAULR_ACTION_URL")

	createMailBox()
	http.HandleFunc("/webhook2email", webHookToEmailHandler)
	err := http.ListenAndServe(":80", nil)
	if nil != err {
		fmt.Println(err)
	}
}
