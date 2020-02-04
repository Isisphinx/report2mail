package main

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/grpc/credentials"
	"net"
	"net/smtp"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/isisphinx/report2mail/proto"
	"github.com/jordan-wright/email"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type reportToEmailServer struct {}

func init() {
	// retrieve config from config file designated by env.var
	configName := os.Getenv("GO_ENV")
	viper.SetConfigName(configName)
	viper.AddConfigPath("./config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	// setup logger config
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("grpc.port")))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("server listening on %d", viper.GetInt("grpc.port"))

	creds, err := credentials.NewServerTLSFromFile("service.pem", "service.key")
	if err != nil {
		log.Fatalf("Failed to setup TLS: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterReportToEmailServer(grpcServer, &reportToEmailServer{})
	// determine whether to use TLS
	grpcServer.Serve(lis)
}

// SendEmail Handle gRPC request
func (server *reportToEmailServer) SendEmail(ctx context.Context, inEmail *proto.EmailToSend) (*proto.SentStatus, error) {

	body, err := generateEmailBody(inEmail)
	if err != nil {
		log.WithField("filename", inEmail.Filename).WithError(err).Error("Failed to generate body")
		return nil, err
	}
	err = handleMailSending(inEmail, body)
	if err != nil {
		log.WithField("filename", inEmail.Filename).WithError(err).Error("Failed to send email")
		return &proto.SentStatus{Status: fmt.Sprintf("Failed to send email: %s", err)}, err
	}
	log.WithField("filename", inEmail.Filename).Infof("OK email sent")
	return &proto.SentStatus{Status: "OK email sent"}, nil
}

// render template into email body
func generateEmailBody(email2send *proto.EmailToSend) (string, error) {
	file, err := os.Open(path.Join("emailTemplates", viper.GetString("email.templateName")))
	if err != nil {
		log.WithError(err).Error("could not open template file")
		return "", err
	}
	tmplt := make([]byte, 200)
	if _, err = file.Read(tmplt); err != nil {
		log.WithError(err).Error("could not read template file")
		return "", err
	}

	t := template.Must(template.New("email").Parse(string(tmplt)))

	formatedDate, err := formatDate(email2send.GetDate())
	if err != nil {
		log.WithField("filename", email2send.Filename).WithError(err).Error("Could not parse date")
	} else {
		email2send.Date = formatedDate
	}

	rendered := bytes.NewBuffer(nil)
	if err = t.Execute(rendered, email2send); err != nil {
		logrus.WithError(err).Error("Cannot execute template")
		return "", err
	}
	return strings.Trim(rendered.String(), "\x00"), nil
}

// actually send the email
func handleMailSending(emailData *proto.EmailToSend, emailBody string) error {
	e := email.NewEmail()
	e.From = viper.GetString("email.sender")
	e.To = []string{emailData.GetEmailAddress()}
	e.Subject = viper.GetString("email.subject")
	e.Text = []byte(emailBody)

	fileBuffer := bytes.NewBuffer(emailData.PdfPayload)
	e.Attach(fileBuffer, emailData.Filename, "application/pdf")

	auth := smtp.PlainAuth("", viper.GetString("smtp.username"), viper.GetString("smtp.password"), viper.GetString("smtp.server"))
	// TODO next line, get from config
	err := e.Send("smtp.gmail.com:587", auth)
	if err != nil {
		log.WithError(err).Error("mail sending failed")
		return err
	}
	return nil
}

func formatDate(indate string) (string, error) {
	var monthInFrench = []string{
		"janvier",
		"février",
		"mars",
		"avril",
		"mai",
		"juin",
		"juillet",
		"aout",
		"septembre",
		"octobre",
		"novembre",
		"décembre",
	}

	informat := "2006-01-02"
	parsed, err := time.Parse(informat, indate)
	if err != nil {
		return "", err
	}
	return strings.Join(
		[]string{parsed.Format("_2"),
			monthInFrench[parsed.Month()-1],
			parsed.Format("2006"),
		}, " "), nil

}
