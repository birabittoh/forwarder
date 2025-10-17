package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/zelenin/go-tdlib/client"
)

type Config struct {
	ApiID                int32
	ApiHash              string
	PhoneNumber          string
	SourceChannelID      int64
	TargetChannelID      int64
	DiscussionGroupID    int64
	CommentTemplate      *client.FormattedText
	CommentNotifications bool
	ShowForwarded        bool
	IgnoreRegex          *regexp.Regexp
	DatabaseDirectory    string
	FilesDirectory       string
	VerbosityLevel       int
}

var (
	cfg         Config
	tdlibClient *client.Client
	authorizer  *ClientAuthorizer
)

type ClientAuthorizer struct {
	PhoneNumber string
}

// Close implements client.AuthorizationStateHandler.
// No cleanup needed.
func (a *ClientAuthorizer) Close() {}

func (a *ClientAuthorizer) Handle(c *client.Client, state client.AuthorizationState) error {
	switch s := state.(type) {
	case *client.AuthorizationStateWaitTdlibParameters:
		_, err := c.SetTdlibParameters(&client.SetTdlibParametersRequest{
			UseTestDc:           false,
			DatabaseDirectory:   cfg.DatabaseDirectory,
			FilesDirectory:      cfg.FilesDirectory,
			UseFileDatabase:     true,
			UseChatInfoDatabase: true,
			UseMessageDatabase:  true,
			UseSecretChats:      false,
			ApiId:               cfg.ApiID,
			ApiHash:             cfg.ApiHash,
			SystemLanguageCode:  "en",
			DeviceModel:         "Server",
			SystemVersion:       "1.0.0",
			ApplicationVersion:  "1.0.0",
		})
		return err

	case *client.AuthorizationStateWaitPhoneNumber:
		_, err := c.SetAuthenticationPhoneNumber(&client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: a.PhoneNumber,
		})
		return err

	case *client.AuthorizationStateWaitCode:
		var code string
		fmt.Print("Enter code: ")
		fmt.Scanln(&code)
		_, err := c.CheckAuthenticationCode(&client.CheckAuthenticationCodeRequest{
			Code: code,
		})
		return err

	case *client.AuthorizationStateWaitPassword:
		var password string
		fmt.Print("Enter password: ")
		fmt.Scanln(&password)
		_, err := c.CheckAuthenticationPassword(&client.CheckAuthenticationPasswordRequest{
			Password: password,
		})
		return err

	case *client.AuthorizationStateReady:
		log.Println("Authorization successful")
		return nil

	default:
		return fmt.Errorf("unsupported authorization state: %v", s.AuthorizationStateType())
	}
}

func loadConfig() error {
	godotenv.Load()

	apiID, err := strconv.Atoi(getEnv("API_ID", ""))
	if err != nil {
		return fmt.Errorf("invalid API_ID: %w", err)
	}
	cfg.ApiID = int32(apiID)

	cfg.ApiHash = getEnv("API_HASH", "")
	cfg.PhoneNumber = getEnv("PHONE_NUMBER", "")

	sourceID, err := strconv.ParseInt(getEnv("SOURCE_CHANNEL_ID", ""), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid SOURCE_CHANNEL_ID: %w", err)
	}
	cfg.SourceChannelID = sourceID

	targetID, err := strconv.ParseInt(getEnv("TARGET_CHANNEL_ID", ""), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid TARGET_CHANNEL_ID: %w", err)
	}
	cfg.TargetChannelID = targetID

	cfg.DiscussionGroupID, _ = strconv.ParseInt(getEnv("DISCUSSION_GROUP_ID", ""), 10, 64)

	// Read comment template from file
	commentBytes, err := os.ReadFile(getEnv("COMMENT_TEMPLATE_FILE", "comment.md"))
	if err == nil {
		cfg.CommentTemplate, err = client.ParseMarkdown(&client.ParseMarkdownRequest{
			Text: &client.FormattedText{
				Text: string(commentBytes),
			},
		})

		if err != nil {
			cfg.CommentTemplate = nil
		}
	}

	cfg.CommentNotifications, err = strconv.ParseBool(getEnv("ENABLE_COMMENT_NOTIFICATIONS", "false"))
	if err != nil {
		cfg.CommentNotifications = false
	}

	cfg.ShowForwarded, err = strconv.ParseBool(getEnv("SHOW_FORWARDED", "false"))
	if err != nil {
		cfg.ShowForwarded = false
	}

	// Compile regex
	ignoreRegexStr := getEnv("IGNORE_REGEX", "")
	if ignoreRegexStr != "" {
		cfg.IgnoreRegex, err = regexp.Compile(ignoreRegexStr)
		if err != nil {
			return fmt.Errorf("invalid IGNORE_REGEX: %w", err)
		}
	}

	cfg.DatabaseDirectory = getEnv("DATABASE_DIRECTORY", "./tdlib-db")
	cfg.FilesDirectory = getEnv("FILES_DIRECTORY", "./tdlib-files")

	cfg.VerbosityLevel, err = strconv.Atoi(getEnv("VERBOSITY_LEVEL", "2"))
	if err != nil {
		return fmt.Errorf("invalid VERBOSITY_LEVEL: %w", err)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initTDLib() error {
	client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{NewVerbosityLevel: int32(cfg.VerbosityLevel)})

	authorizer = &ClientAuthorizer{
		PhoneNumber: cfg.PhoneNumber,
	}

	var err error
	tdlibClient, err = client.NewClient(authorizer)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return nil
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

func shouldForwardMessage(text *client.FormattedText) bool {
	if text == nil {
		return false
	}

	if cfg.IgnoreRegex == nil {
		return true // no regex = no check
	}

	return !cfg.IgnoreRegex.MatchString(text.Text)
}

func forwardMessage(messageID int64) error {
	// Forward message to target channel
	_, err := tdlibClient.ForwardMessages(&client.ForwardMessagesRequest{
		ChatId:        cfg.TargetChannelID,
		FromChatId:    cfg.SourceChannelID,
		MessageIds:    []int64{messageID},
		SendCopy:      cfg.ShowForwarded,
		RemoveCaption: false,
	})

	if err != nil {
		return err
	}

	log.Printf("Message %d forwarded successfully", messageID)
	return nil
}

func postComment(message *client.Message) error {
	_, err := tdlibClient.SendMessage(&client.SendMessageRequest{
		ChatId:              cfg.DiscussionGroupID,
		MessageThreadId:     message.MessageThreadId,
		ReplyTo:             &client.InputMessageReplyToMessage{MessageId: message.Id},
		InputMessageContent: &client.InputMessageText{Text: cfg.CommentTemplate},
		Options:             &client.MessageSendOptions{DisableNotification: cfg.CommentNotifications},
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

func shouldPostComment(message *client.Message) bool { // false = send comment, true = ignore
	if cfg.CommentTemplate == nil {
		return false
	}

	// Don't send again if the message's content is equal to the comment template
	if compareMessages(getMessageText(message.Content), cfg.CommentTemplate) {
		return false
	}

	switch sender := message.SenderId.(type) {
	case *client.MessageSenderUser:
		return false
	case *client.MessageSenderChat:
		return sender.ChatId == cfg.TargetChannelID
	}
	return false
}

func handleUpdate(update client.Update) {
	switch u := update.(type) {
	case *client.UpdateNewMessage:

		switch u.Message.ChatId {

		case cfg.SourceChannelID:
			if !shouldForwardMessage(getMessageText(u.Message.Content)) { // Check if message should be forwarded
				log.Printf("Message %d was not forwarded", u.Message.Id)
				return
			}

			log.Printf("New message from source channel: %d", u.Message.Id)

			// Forward message
			if err := forwardMessage(u.Message.Id); err != nil {
				log.Printf("Error forwarding message: %v", err)
				return
			}

		case cfg.DiscussionGroupID: // will be 0 if not set
			if !shouldPostComment(u.Message) { // Check if comment should be posted
				return
			}

			log.Printf("Message %d is valid, posting comment", u.Message.Id)

			// Post comment in discussion group
			if err := postComment(u.Message); err != nil {
				log.Printf("Error posting comment: %v", err)
			}
		}
	}
}

func main() {
	log.Println("Starting Telegram Forwarder Bot...")

	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("- Source Channel: %d", cfg.SourceChannelID)
	log.Printf("- Target Channel: %d", cfg.TargetChannelID)
	log.Printf("- Discussion Group: %d", cfg.DiscussionGroupID)
	log.Printf("- Ignore Regex: %s", cfg.IgnoreRegex)

	// Initialize TDLib
	if err := initTDLib(); err != nil {
		log.Fatalf("Failed to initialize TDLib: %v", err)
	}

	log.Println("TDLib initialized successfully")

	// Listen for updates
	listener := tdlibClient.GetListener()
	defer listener.Close()

	log.Println("Listening for updates...")

	for update := range listener.Updates {
		if update.GetClass() == client.ClassUpdate {
			if upd, ok := update.(client.Update); ok {
				handleUpdate(upd)
			}
		}
	}
}
