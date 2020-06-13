package mail

import (
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/jace-ys/spautofy/pkg/users"
)

type Mailer interface {
	SendNewPlaylistEmail(user *users.User, withConfirm bool, playlistURL string) error
}

type SendGridConfig struct {
	APIKey      string
	SenderName  string
	SenderEmail string
	TemplateID  string
}

type SendGridMailer struct {
	client      *sendgrid.Client
	senderName  string
	senderEmail string
	templateID  string
}

func NewSendGridMailer(cfg *SendGridConfig) *SendGridMailer {
	return &SendGridMailer{
		client:      sendgrid.NewSendClient(cfg.APIKey),
		senderName:  cfg.SenderName,
		senderEmail: cfg.SenderEmail,
		templateID:  cfg.TemplateID,
	}
}

func (m *SendGridMailer) SendNewPlaylistEmail(user *users.User, withConfirm bool, playlistURL string) error {
	email := mail.NewV3Mail()

	p := mail.NewPersonalization()
	p.AddTos(mail.NewEmail(user.DisplayName, user.Email))

	email.SetFrom(mail.NewEmail(m.senderName, m.senderEmail))
	email.AddPersonalizations(p)

	var firstName string
	name := strings.SplitN(user.DisplayName, " ", 2)
	if len(name) > 1 {
		firstName = name[0]
	} else {
		firstName = user.DisplayName
	}

	email.SetTemplateID(m.templateID)
	p.SetDynamicTemplateData("firstName", firstName)
	p.SetDynamicTemplateData("withConfirm", withConfirm)
	p.SetDynamicTemplateData("playlistLink", playlistURL)

	_, err := m.client.Send(email)
	if err != nil {
		return err
	}

	return nil
}
