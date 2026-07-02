package models

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

var b *tb.Bot
var tgg *tb.Chat

func initTgBot() {
	go func() {
		if Config.TelegramBotToken == "" {
			return
		}
		var err error
		b, err = tb.NewBot(tb.Settings{
			URL:    Config.TGURL,
			Token:  Config.TelegramBotToken,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			Bot().Warnf("监听tgbot失败")
			Bot().Infof("%v", err)
			return
		}
		handle := func(m *tb.Message) {
			if !m.FromGroup() {
				rt := handleMessage(m.Text, "tg", m.Sender.ID)
				switch rt.(type) {
				case string:
					b.Send(m.Sender, rt.(string))
				case *http.Response:
					b.SendAlbum(m.Sender, tb.Album{&tb.Photo{File: tb.FromReader(rt.(*http.Response).Body)}})
				}
			} else {
				if tgg == nil {
					tgg = m.Chat
				}
				var rid int = 0
				if m.ReplyTo != nil {
					rid = m.ReplyTo.Sender.ID
				}

				rt := handleMessage(m.Text, "tgg", m.Sender.ID, int(m.Chat.ID), m.ID, m.Sender.Username, rid)
				switch rt.(type) {
				case string:
					b.Send(m.Chat, rt.(string), &tb.SendOptions{ReplyTo: m})
				case *http.Response:
					b.SendAlbum(m.Chat, tb.Album{&tb.Photo{File: tb.FromReader(rt.(*http.Response).Body)}}, &tb.SendOptions{ReplyTo: m})
				}
			}
		}

		b.Handle(tb.OnDocument, func(m *tb.Message) {
			if m.Sender.ID != Config.TelegramUserID {
				return
			}
			if regexp.MustCompile(`.js$`).FindString(m.Document.FileName) == "" && regexp.MustCompile(`.py$`).FindString(m.Document.FileName) == "" {
				return
			}
			b.Download(m.Document.MediaFile(), ExecPath+"/scripts/"+m.Document.FileName)
			m.Text = fmt.Sprintf("run " + m.Document.FileName)
			handle(m)
		})
		b.Handle(tb.OnText, handle)
		Bot().Infof("监听tgbot")
		b.Start()
	}()
}

func SendTgMsg(uid int, msg string) {
	if b == nil || uid == 0 {
		return
	}
	b.Send(&tb.User{ID: uid}, msg)
}

func SendTgImg(uid int, file []byte) {
	unix := time.Now().Unix()

	filename := ExecPath + fmt.Sprintf("/static/%d.jpg", unix)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		Bot().Warnf("zqdyj.txt失败，%v", err)
	}
	f.Write(file)
	f.Close()

	img := uploadImg(filename)
	Bot().Infof("%s", img)

	os.Remove(filename)

	if b == nil || uid == 0 {
		return
	}
	b.Send(&tb.User{ID: uid}, &tb.Photo{
		File: tb.File{
			FileURL: img,
		},
	})
}

func SendTggMsg(gid int, uid int, msg string, mid int, unm string) {
	if b == nil || uid == 0 {
		return
	}
	if unm != "" {
		b.Send(&tb.Chat{ID: int64(gid)}, fmt.Sprintf("@%s %s", unm, msg))
	} else {
		b.Send(&tb.Chat{ID: int64(gid)}, msg, &tb.SendOptions{ReplyTo: &tb.Message{ID: mid}})
	}

}

func SendTggImg(gid int, uid int, file []byte, mid int, unm string) {
	unix := time.Now().Unix()

	filename := ExecPath + fmt.Sprintf("/static/%d.jpg", unix)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		Bot().Warnf("zqdyj.txt失败，%v", err)
	}
	f.Write(file)
	f.Close()

	img := uploadImg(filename)
	Bot().Infof("%s", img)

	os.Remove(filename)

	if b == nil || uid == 0 {
		return
	}
	if unm != "" {
		b.Send(&tb.Chat{ID: int64(gid)}, &tb.Photo{
			File: tb.File{
				FileURL: img,
			},
			Caption: fmt.Sprintf("@%s", unm),
		})
	} else {
		b.Send(&tb.Chat{ID: int64(gid)}, &tb.Photo{
			File: tb.File{
				FileURL: img,
			},
		}, &tb.SendOptions{ReplyTo: &tb.Message{ID: mid}})
	}

}

func SendTgImg2(uid int, msg string) {
	
	if b == nil || uid == 0 {
		return
	}
    Bot().Infof("%s", msg)
    b.Send(&tb.User{ID: uid}, &tb.Photo{
		File: tb.File{
			FileURL: msg,
		},
	})
}

func SendTggImg2(gid int, uid int, msg string, mid int, unm string) {

	if b == nil || uid == 0 {
		return
	}
	if unm != "" {
		b.Send(&tb.Chat{ID: int64(gid)}, &tb.Photo{
			File: tb.File{
				FileURL: msg,
			},
			Caption: fmt.Sprintf("@%s", unm),
		})
	} else {
		b.Send(&tb.Chat{ID: int64(gid)}, &tb.Photo{
			File: tb.File{
				FileURL: msg,
			},
		}, &tb.SendOptions{ReplyTo: &tb.Message{ID: mid}})
	}

}
