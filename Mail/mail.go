package Mail

import (
	"bytes"
	"io"
	"log"
	"sync"
	"fmt"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"strings"
	"github.com/emersion/go-message"
	"mime/quotedprintable"

)

type Mail struct {
	ID   string
	From string
	To   []string
	Data string
}

type MailList struct {
	mu    sync.Mutex
	Mails []Mail
}

func NewMailList() *MailList {
	return &MailList{Mails: []Mail{}}
}

func (ml *MailList) Add(mail Mail) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.Mails = append(ml.Mails, mail)
}

func (ml *MailList) GetAll() []Mail {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return append([]Mail{}, ml.Mails...) // copie défensive
}

func (ml *MailList) Flush() {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.Mails = []Mail{}
}

func (ml *MailList) DeleteByID(id string) bool {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	for i, m := range ml.Mails {
		if m.ID == id {
			ml.Mails = append(ml.Mails[:i], ml.Mails[i+1:]...)
			return true
		}
	}
	return false
}

// ─────── SMTP BACKEND ───────

type Backend struct {
	store *MailList
}

func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		store: bkd.store,
	}, nil
}

type Session struct {
	from  string
	to    []string
	data  bytes.Buffer
	store *MailList
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	log.Println("[+] Mail received from:", from)

	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	s.data.Reset()
	_, err := io.Copy(&s.data, r)
	if err != nil {
		return err
	}

	// Ajout à la liste centralisée
	text,err := extractPlainTextEmail(s.data.Bytes())
	if err != nil {
		text = s.data.String()
	} else {
		text = decodeQuotedPrintable(text)
	}
	m := Mail{
		ID:   uuid.NewString(),
		From: s.from,
		To:   s.to,
		Data: text,
	}
	s.store.Add(m)
	return nil
}

func (s *Session) Reset()         {}
func (s *Session) Logout() error  { return nil }

// ─────── Lancement Serveur ───────

func InitServer(domain string, store *MailList) *smtp.Server {
	be := &Backend{store: store}
	s := smtp.NewServer(be)
	s.Addr = ":25"
	s.Domain = domain
	s.AllowInsecureAuth = true
	return s
}

func StartServer(s *smtp.Server) {
	log.Println("[+] SMTP server started", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal("Erreur SMTP :", err)
	}
}


func extractPlainTextEmail(raw []byte) (string, error) {
	// Lire le message MIME complet
	entity, err := message.Read(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("échec de lecture du message MIME: %w", err)
	}

	mediaType, _, err := entity.Header.ContentType()
	if err != nil {
		body, _ := io.ReadAll(entity.Body)
		return string(body), nil
	}

	// Si multipart, utiliser .MultipartReader()
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := entity.MultipartReader()
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			pType, _, _ := part.Header.ContentType()
			if pType == "text/plain" {
				data, _ := io.ReadAll(part.Body)
				return string(data), nil
			}
		}
	}

	// Sinon, juste lire le corps
	data, _ := io.ReadAll(entity.Body)
	return string(data), nil
}
func decodeQuotedPrintable(input string) (string) {
	reader := quotedprintable.NewReader(strings.NewReader(input))
	buf := new(bytes.Buffer)
	io.Copy(buf, reader)
	return buf.String()
}