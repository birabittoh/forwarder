package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/zelenin/go-tdlib/client"
)

type Config struct {
	ApiID             int32
	ApiHash           string
	PhoneNumber       string
	SourceChannelID   int64
	TargetChannelID   int64
	DiscussionGroupID int64
	IgnoreRegex       string
	CommentTemplate   string
	DatabaseDirectory string
	FilesDirectory    string
	VerbosityLevel    int
}

var (
	cfg           Config
	ignorePattern *regexp.Regexp
	tdlibClient   *client.Client
	authorizer    *ClientAuthorizer
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

	discussionID, err := strconv.ParseInt(getEnv("DISCUSSION_GROUP_ID", ""), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid DISCUSSION_GROUP_ID: %w", err)
	}
	cfg.DiscussionGroupID = discussionID

	cfg.IgnoreRegex = getEnv("IGNORE_REGEX", "#aff")
	cfg.DatabaseDirectory = getEnv("DATABASE_DIRECTORY", "./tdlib-db")
	cfg.FilesDirectory = getEnv("FILES_DIRECTORY", "./tdlib-files")

	cfg.VerbosityLevel, err = strconv.Atoi(getEnv("VERBOSITY_LEVEL", "2"))
	if err != nil {
		return fmt.Errorf("invalid VERBOSITY_LEVEL: %w", err)
	}

	// Compile regex
	ignorePattern, err = regexp.Compile(cfg.IgnoreRegex)
	if err != nil {
		return fmt.Errorf("invalid IGNORE_REGEX: %w", err)
	}

	// Read comment template from file
	commentBytes, err := os.ReadFile(getEnv("COMMENT_TEMPLATE_FILE", "comment.md"))
	if err != nil {
		return fmt.Errorf("failed to read COMMENT_TEMPLATE_FILE: %w", err)
	}
	cfg.CommentTemplate = strings.TrimSpace(string(commentBytes))

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

func getMessageText(content client.MessageContent) string {
	switch c := content.(type) {
	case *client.MessageText:
		return c.Text.Text
	case *client.MessagePhoto:
		return c.Caption.Text
	case *client.MessageVideo:
		return c.Caption.Text
	case *client.MessageDocument:
		return c.Caption.Text
	default:
		return ""
	}
}

func shouldIgnoreMessage(text string) bool {
	if text == "" {
		return false
	}
	return ignorePattern.MatchString(text)
}

func forwardMessage(messageID int64) (m *client.Messages, err error) {
	// Forward message to target channel
	m, err = tdlibClient.ForwardMessages(&client.ForwardMessagesRequest{
		ChatId:        cfg.TargetChannelID,
		FromChatId:    cfg.SourceChannelID,
		MessageIds:    []int64{messageID},
		SendCopy:      true,
		RemoveCaption: false,
	})

	if err == nil {
		log.Printf("Message %d forwarded successfully", messageID)
	}

	return
}

func postComment(message *client.Message) error {
	commentText := &client.FormattedText{
		Text:     cfg.CommentTemplate,
		Entities: []*client.TextEntity{},
	}

	// Parse markdown if needed
	parsedText, err := tdlibClient.ParseMarkdown(&client.ParseMarkdownRequest{
		Text: commentText,
	})
	if err != nil {
		log.Printf("Warning: failed to parse markdown: %v", err)
	} else {
		commentText = parsedText
	}

	inputContent := &client.InputMessageText{
		Text:       commentText,
		ClearDraft: false,
	}

	_, err = tdlibClient.SendMessage(&client.SendMessageRequest{
		ChatId:              cfg.DiscussionGroupID,
		MessageThreadId:     message.MessageThreadId,
		ReplyTo:             &client.InputMessageReplyToMessage{MessageId: message.Id},
		InputMessageContent: inputContent,
	})

	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	log.Printf("Comment posted for message %d", message.Id)
	return nil
}

func isMessageFromMe(message *client.Message) bool {
	// Check if the message's content is equal to the comment template
	messageText := getMessageText(message.Content)
	if messageText == cfg.CommentTemplate {
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
			// Check if message should be ignored
			if shouldIgnoreMessage(getMessageText(u.Message.Content)) {
				log.Printf("Message %d ignored (matched regex)", u.Message.Id)
				return
			}

			log.Printf("New message from source channel: %d", u.Message.Id)

			// Forward message
			_, err := forwardMessage(u.Message.Id)
			if err != nil {
				log.Printf("Error forwarding message: %v", err)
				return
			}

		case cfg.DiscussionGroupID:
			// Verifica se il messaggio è dal nostro account
			if !isMessageFromMe(u.Message) {
				return
			}

			log.Printf("Message %d is from our account, posting comment", u.Message.Id)

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
