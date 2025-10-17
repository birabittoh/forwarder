package forwarder

import (
	"fmt"
	"log"

	"github.com/birabittoh/forwarder/config"
	"github.com/zelenin/go-tdlib/client"
)

type Forwarder struct {
	cfg         *config.Config
	tdlibClient *client.Client
}

func New(cfg *config.Config) (*Forwarder, error) {
	client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{NewVerbosityLevel: int32(cfg.VerbosityLevel)})

	forwarder := &Forwarder{cfg: cfg}
	authorizer := &ClientAuthorizer{cfg: cfg}

	var err error
	forwarder.tdlibClient, err = client.NewClient(authorizer)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return forwarder, nil
}

func (f *Forwarder) Listen() {
	listener := f.tdlibClient.GetListener()
	defer listener.Close()

	log.Println("Listening for updates...")

	for update := range listener.Updates {
		if update.GetClass() == client.ClassUpdate {
			if upd, ok := update.(client.Update); ok {
				f.handleUpdate(upd)
			}
		}
	}
}

func (f *Forwarder) shouldForwardMessage(text *client.FormattedText) bool {
	if text == nil {
		return false
	}

	if f.cfg.IgnoreRegex == nil {
		return true // no regex = no check
	}

	return !f.cfg.IgnoreRegex.MatchString(text.Text)
}

func (f *Forwarder) forwardMessage(messageID int64) error {
	// Forward message to target channel
	_, err := f.tdlibClient.ForwardMessages(&client.ForwardMessagesRequest{
		ChatId:        f.cfg.TargetChannelID,
		FromChatId:    f.cfg.SourceChannelID,
		MessageIds:    []int64{messageID},
		SendCopy:      !f.cfg.ShowForwarded,
		RemoveCaption: false,
	})

	if err != nil {
		return err
	}

	log.Printf("Message %d forwarded successfully", messageID)
	return nil
}

func (f *Forwarder) postComment(message *client.Message) error {
	_, err := f.tdlibClient.SendMessage(&client.SendMessageRequest{
		ChatId:              f.cfg.DiscussionGroupID,
		MessageThreadId:     message.MessageThreadId,
		ReplyTo:             &client.InputMessageReplyToMessage{MessageId: message.Id},
		InputMessageContent: &client.InputMessageText{Text: f.cfg.CommentTemplate},
		Options:             &client.MessageSendOptions{DisableNotification: f.cfg.CommentNotifications},
	})

	if err != nil {
		return err
	}

	log.Printf("Comment posted for message %d", message.Id)
	return nil
}

func compareMessages(msg1, msg2 *client.FormattedText) bool {
	if msg1 == nil || msg2 == nil {
		return false
	}

	if msg1.Type != msg2.Type {
		return false
	}

	if msg1.Text != msg2.Text {
		return false
	}

	if msg1.Extra != msg2.Extra {
		return false
	}

	if len(msg1.Entities) != len(msg2.Entities) {
		return false
	}

	for i := range msg1.Entities {
		ent1 := msg1.Entities[i]
		ent2 := msg2.Entities[i]

		if ent1.Offset != ent2.Offset || ent1.Length != ent2.Length || ent1.Type != ent2.Type {
			return false
		}
	}

	return true
}

func (f *Forwarder) shouldPostComment(message *client.Message) bool { // false = send comment, true = ignore
	if f.cfg.CommentTemplate == nil {
		return false
	}

	// Don't send again if the message's content is equal to the comment template
	if compareMessages(getMessageText(message.Content), f.cfg.CommentTemplate) {
		return false
	}

	switch sender := message.SenderId.(type) {
	case *client.MessageSenderUser:
		return false
	case *client.MessageSenderChat:
		return sender.ChatId == f.cfg.TargetChannelID
	}
	return false
}

func (f *Forwarder) handleUpdate(update client.Update) {
	switch u := update.(type) {
	case *client.UpdateNewMessage:

		switch u.Message.ChatId {

		case f.cfg.SourceChannelID:
			if !f.shouldForwardMessage(getMessageText(u.Message.Content)) { // Check if message should be forwarded
				log.Printf("Message %d was not forwarded", u.Message.Id)
				return
			}

			log.Printf("New message from source channel: %d", u.Message.Id)

			// Forward message
			if err := f.forwardMessage(u.Message.Id); err != nil {
				log.Printf("Error forwarding message: %v", err)
				return
			}

		case f.cfg.DiscussionGroupID: // will be 0 if not set
			if !f.shouldPostComment(u.Message) { // Check if comment should be posted
				return
			}

			log.Printf("Message %d is valid, posting comment", u.Message.Id)

			// Post comment in discussion group
			if err := f.postComment(u.Message); err != nil {
				log.Printf("Error posting comment: %v", err)
			}
		}
	}
}

func getMessageText(content client.MessageContent) *client.FormattedText {
	switch c := content.(type) {
	case *client.MessageText:
		return c.Text
	case *client.MessagePhoto:
		return c.Caption
	case *client.MessageVideo:
		return c.Caption
	case *client.MessageDocument:
		return c.Caption
	default:
		return nil
	}
}
