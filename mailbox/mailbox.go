package mailbox

import (
	"fmt"
	"io"
	"log"
	"mime"
	"regexp"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

type NewMessageEvent = func(title, message, actionUrl string)

type IMAPClient struct {
	IDleClient *imapclient.Client
	Client     *imapclient.Client
	NewMessage chan uint32
}

var iClient *IMAPClient

// / 邮箱收到新的邮件
func mailBoxNewMessage(data *imapclient.UnilateralDataMailbox) {
	if data.NumMessages != nil && nil != iClient {
		log.Println("邮件接受客户端 '接收到了新' 消息", data.NumMessages)
		if data.NumMessages != nil {
			iClient.NewMessage <- *data.NumMessages
		}
	}
}

func NewClient(host, port string) (*IMAPClient, error) {
	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Expunge: func(seqNum uint32) {
				log.Printf("message %v has been expunged", seqNum)
			},
			Mailbox: mailBoxNewMessage,
			Fetch: func(msg *imapclient.FetchMessageData) {
				log.Printf("邮件接受客户端 'Fetch' 消息")
			},
		},
	}

	idleClient, err := imapclient.DialTLS(fmt.Sprintf("%s:%s", host, port), options)
	if nil != err {
		fmt.Println("邮件监听客户端 '连接' 异常了：", err)
		return nil, err
	}

	if client, err := imapclient.DialTLS(fmt.Sprintf("%s:%s", host, port), options); nil != err {
		fmt.Println("邮件接受客户端 '连接' 异常了：", err)
		return nil, err
	} else {
		fmt.Println("创建邮箱客户端成功", err)
		iClient = &IMAPClient{IDleClient: idleClient, Client: client, NewMessage: make(chan uint32, 10)}
	}
	return iClient, nil
}

func (iClient *IMAPClient) Login(from, password string) error {
	if err := iClient.IDleClient.Login(from, password).Wait(); nil != err {
		fmt.Println("邮件监听客户端 '登陆' 异常了：", err)
		return err
	}
	if err := iClient.IDleClient.Select("INBOX", nil); err != nil {
		fmt.Println("邮件监听客户端 '进入邮箱' 了：")
	}
	if err := iClient.Client.Login(from, password).Wait(); nil != err {
		fmt.Println("邮件接受客户端 '登陆' 异常了：", err)
		return err
	}
	if err := iClient.Client.Select("INBOX", nil); err != nil {
		fmt.Println("邮件接受客户端 '进入邮箱' 了：")
	}
	return nil
}

func (iClient *IMAPClient) Idle(event NewMessageEvent) {
	if _, err := iClient.IDleClient.Idle(); nil != err {
		fmt.Println("邮件接收监听失败", err)
	} else {
		for {
			i, ok := <-iClient.NewMessage
			if ok {
				iClient.parseEmailOfMessage(i, event)
			}
		}
	}

}

func (iClient *IMAPClient) parseEmailOfMessage(numMessages uint32, event NewMessageEvent) {
	seqSet := imap.SeqSetNum(numMessages)
	if nil == seqSet {
		return
	}
	fetchOptions := &imap.FetchOptions{Envelope: true, UID: true, BodySection: []*imap.FetchItemBodySection{{}}}
	msg := iClient.Client.Fetch(seqSet, fetchOptions)
	defer msg.Close()
	msgCmd := msg.Next()
	if nil == msgCmd {
		fmt.Println("为什么会是空的？？？？？")
		return
	}

	var bodySection imapclient.FetchItemDataBodySection
	ok := false
	for {
		item := msgCmd.Next()
		if item == nil {
			break
		}
		bodySection, ok = item.(imapclient.FetchItemDataBodySection)
		if ok {
			break
		}
	}
	if !ok {
		log.Fatalf("FETCH command did not return body section")
	}
	mr, err := mail.CreateReader(bodySection.Literal)
	if err != nil {
		log.Fatalf("failed to create mail reader: %v", err)
	}
	h := mr.Header
	if date, err := h.Date(); err != nil {
		log.Printf("failed to parse Date header field: %v", err)
	} else {
		log.Printf("Date: %v", date)
	}
	if to, err := h.AddressList("To"); err != nil {
		log.Printf("failed to parse To header field: %v", err)
	} else {
		log.Printf("To: %v", to)
	}
	var title string
	if subject, err := h.Text("Subject"); err != nil {
		log.Printf("failed to parse Subject header field: %v", err)
		title = ""
	} else {
		log.Printf("Subject: %v", subject)
		title = subject
	}

	var message string
	// Process the message's parts
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("failed to read message part: %v", err)
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := io.ReadAll(p.Body)
			message = string(b)
			log.Printf("Inline text: %v", message)
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			log.Printf("Attachment: %v", filename)
		}
	}
	fmt.Println("拿到信息了 title=", title, "message=", message)
	patt := `https?://[a-zA-Z0-9.-]+(/S+)?`
	re := regexp.MustCompile(patt)
	urls := re.FindAllString(message, -1)
	var url string
	if len(urls) > 0 {
		url = urls[0]
	}

	event(title, message, url)
	msg.Close()
}