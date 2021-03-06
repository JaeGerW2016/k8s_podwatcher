package email

import (
	"bytes"
	"flag"
	"fmt"
	"gopkg.in/gomail.v1"
	"html/template"
	"k8s_podwatcher/pkg/handlers"
	"strings"
	"time"
)

var (
	smtpHost      string
	smtpPort      int
	smtpUsername  string
	smtpPassword  string
	smtpReceivers string
)

func init() {
	flag.StringVar(&smtpHost, "smtp-host", "", "smtp server host")
	flag.IntVar(&smtpPort, "smtp-port", 446, "smtp server port")
	flag.StringVar(&smtpUsername, "smtp-username", "", "username")
	flag.StringVar(&smtpPassword, "smtp-password", "", "password")
	flag.StringVar(&smtpReceivers, "smtp-receivers", "", "receivers")
}

type handler struct {
}

func NewHaddler() handlers.Handler {
	return (*handler)(nil)
}

var emailTemplate = template.Must(template.New("").Parse(`<html>
	<body>
		Container {{ .ContainerName }} in pod {{ .Namespace }}/{{ .Name }} is crashed<br>
		<h1>Reason:</h1>{{ .Reason }}
		<h1>Message:</h1>{{ .Message }}
		<h1>Logs:</h1>
		{{range .RawLogs}}<div>{{ . }}</div>{{else}}<div><strong>no logs</strong></div>{{end}}
	</body>
</html>`))

func (*handler) Handle(event *handlers.Event) error {
	message := gomail.NewMessage()
	message.SetHeader("From", smtpUsername)
	message.SetHeader("To", strings.Split(smtpReceivers, ",")...)
	message.SetHeader("Subject", fmt.Sprintf("Container %s in pod %s/%s crashed", event.ContainerName, event.Namespace, event.Name))
	message.SetDateHeader("Date", time.Now())

	buf := bytes.NewBuffer(nil)
	if err := emailTemplate.Execute(buf, event); err != nil {
		return fmt.Errorf("render template failed: %v", err)
	}
	message.SetBody("text/html", buf.String())

	d :=gomail.NewDialer(smtpHost,smtpPort,smtpUsername,smtpPassword)
	if err := d.DialAndSend(message); err != nil {
		fmt.Println(err)
	}
	return nil
}
