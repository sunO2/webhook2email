package mailbox

import (
	"fmt"
	"io"
	"log"
	"mime"
	"regexp"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

type NewMessageEvent = func(title, message, actionUrl string)

// idle 重新启动时间
var reStartIdle = 5 * time.Minute

type IMAPClient struct {
	Client     *imapclient.Client
	NewMessage chan *uint32
	host       string
	port       string
	password   string
	from       string
}

var iClient *IMAPClient = &IMAPClient{NewMessage: make(chan *uint32)}

// / 邮箱收到新的邮件
func mailBoxNewMessage(data *imapclient.UnilateralDataMailbox) {
	if data.NumMessages != nil && nil != iClient {
		log.Println("邮件接受客户端 '接收到了新' 消息", data.NumMessages)
		if data.NumMessages != nil {
			iClient.NewMessage <- data.NumMessages
		}
	}
}

func NewClient(host, port string) (*IMAPClient, error) {
	options := &imapclient.Options{
		DebugWriter: log.Default().Writer(),
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

	if client, err := imapclient.DialTLS(fmt.Sprintf("%s:%s", host, port), options); nil != err {
		log.Println("邮件接受客户端 '连接' 异常了：", err)
		return nil, err
	} else {
		log.Println("创建邮箱客户端成功", err)
		iClient.Client = client
	}
	iClient.host = host
	iClient.port = port
	return iClient, nil
}

func (iClient *IMAPClient) close() {
	iClient.Client.Close()
}

func (iClient *IMAPClient) reConnect() {
	NewClient(iClient.host, iClient.port)
	iClient.Login(iClient.from, iClient.password)
}

func (iClient *IMAPClient) Login(from, password string) error {
	if err := iClient.Client.Login(from, password).Wait(); nil != err {
		log.Println("邮件接受客户端 '登陆' 异常了：", err)
		return err
	}
	if err := iClient.Client.Select("INBOX", nil); err != nil {
		log.Println("邮件接受客户端 '进入邮箱' 了：")
	}
	iClient.from = from
	iClient.password = password
	return nil
}

func (iClient *IMAPClient) Idle(event NewMessageEvent) {
	wait := func() {
		time.Sleep(5 * time.Second)
	}

	go func(c *IMAPClient) {
		ticker := time.NewTicker(reStartIdle)
		defer ticker.Stop()

		for loop := true; loop; {
			idle, err := c.Client.Idle()
			if err != nil {
				log.Println("定时 Idle 异常", err)
				c.close()
				loop = false
			}

			var messageNum *uint32
			select {
			case messageNum = <-c.NewMessage:
				log.Println("收到消息了。。。。。。。。", *messageNum)
				ticker.Reset(reStartIdle)
			case <-ticker.C:
				{

				}

			}
			// 休眠能解决 ？？？？
			err = idle.Close()
			if err != nil {
				log.Println("定时 Close 异常", err)
				wait()
			}

			err = idle.Wait()
			if err != nil {
				log.Println("定时 Wait 异常", err)
				wait()
				loop = false
			}
			if nil != messageNum {
				wait()
				c.parseEmailOfMessage(*messageNum, event)
			} else {
				wait()
			}
		}
		c.close()
		c.reConnect()
		c.Idle(event)
	}(iClient)
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
		log.Println("为什么会是空的？？？？？")
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
	if mr, err := mail.CreateReader(bodySection.Literal); nil != err {
		log.Fatalf("failed to create mail reader: %v", err)
	} else {
		defer mr.Close()
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
			// log.Printf("Subject: %v", subject)
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
				// log.Printf("Inline text: %v", message)
			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Printf("Attachment: %v", filename)
			}
		}
		patt := `https?://[a-zA-Z0-9.-]+(/S+)?`
		re := regexp.MustCompile(patt)
		urls := re.FindAllString(message, -1)
		var url string
		if len(urls) > 0 {
			url = urls[0]
		}

		event(title, message, url)
	}

}
