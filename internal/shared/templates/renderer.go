package templates

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed *.html
var templateFiles embed.FS

// EmailRenderer es el encargado de cargar y renderizar las plantillas HTML
type EmailRenderer struct {
	templates *template.Template
}

// NewEmailRenderer inicializa el renderizador leyendo los archivos incrustados
func NewEmailRenderer() (*EmailRenderer, error) {
	tmpl, err := template.ParseFS(templateFiles, "*.html")
	if err != nil {
		return nil, err
	}
	return &EmailRenderer{templates: tmpl}, nil
}

// Estructuras de datos para cada tipo de correo

type PasswordResetData struct {
	Name      string
	ResetLink string
}

type ConfirmAccountData struct {
	Name        string
	ConfirmLink string
}

type WelcomeData struct {
	Name      string
	LoginLink string
}

type InvitePostData struct {
	Name        string
	InviterName string
	PostTitle   string
	InviteLink  string
}


func (r *EmailRenderer) RenderPasswordReset(data PasswordResetData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "password_reset.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderConfirmAccount(data ConfirmAccountData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "confirm_account.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderWelcome(data WelcomeData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "welcome.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderInvitePost(data InvitePostData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "invite_post.html", data)
	return body.String(), err
}
