package forwarder

import (
	"fmt"
	"log"

	"github.com/birabittoh/forwarder/config"
	"github.com/zelenin/go-tdlib/client"
)

type ClientAuthorizer struct {
	cfg *config.Config
}

// Close implements client.AuthorizationStateHandler.
// No cleanup needed.
func (a *ClientAuthorizer) Close() {}

func (a *ClientAuthorizer) Handle(c *client.Client, state client.AuthorizationState) error {
	switch s := state.(type) {
	case *client.AuthorizationStateWaitTdlibParameters:
		_, err := c.SetTdlibParameters(&client.SetTdlibParametersRequest{
			UseTestDc:           false,
			DatabaseDirectory:   a.cfg.DatabaseDirectory,
			FilesDirectory:      a.cfg.FilesDirectory,
			UseFileDatabase:     true,
			UseChatInfoDatabase: true,
			UseMessageDatabase:  true,
			UseSecretChats:      false,
			ApiId:               a.cfg.ApiID,
			ApiHash:             a.cfg.ApiHash,
			SystemLanguageCode:  "en",
			DeviceModel:         "Server",
			SystemVersion:       "1.0.0",
			ApplicationVersion:  "1.0.0",
		})
		return err

	case *client.AuthorizationStateWaitPhoneNumber:
		_, err := c.SetAuthenticationPhoneNumber(&client.SetAuthenticationPhoneNumberRequest{
			PhoneNumber: a.cfg.PhoneNumber,
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
