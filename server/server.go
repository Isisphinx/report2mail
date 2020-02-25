package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/go-playground/validator.v9"
)

type reportToEmailServer struct {
	Conf config
}

type config struct {
	SMTP struct {
		Server   string `validate:"required"`
		Port     string `validate:"required"`
		Username string `validate:"required"`
		Password string `validate:"required"`
	}
	GRPCport int `validate:"required"`
	Tokens map[string]string `validate:"required"`
	Email    struct {
		Sender       string `validate:"required"`
		Subject      string `validate:"required"`
		TemplateName string `validate:"required"`
	}
}

var server reportToEmailServer

func init() {
	// setup logger config
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// retrieve config from config file designated by env.var
	configName := os.Getenv("GO_ENV")
	viper.SetConfigName(configName)
	viper.AddConfigPath("./config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	switch err.(type) {
	case nil:
		break
	case viper.ConfigFileNotFoundError:
		log.Debug("No config file found.")
	default:
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	conf := config{}
	viper.BindEnv("PORT")
	conf.GRPCport = viper.GetInt("PORT")
	viper.UnmarshalExact(&conf)

	viper.SetEnvPrefix("R2M")
	viper.BindEnv("conf")
	if len(viper.GetString("conf")) > 0 {
		err = json.Unmarshal([]byte(viper.GetString("conf")), &conf)
		if err != nil {
			panic(err)
		}
	}
	tokens := make(map[string]string, 1)
	viper.BindEnv("tokens")
	if len(viper.GetString("tokens")) > 0 {
		err = json.Unmarshal([]byte(viper.GetString("tokens")), &tokens)
		if err != nil {
			panic(err)
		}
		conf.Tokens = tokens
	}

	err = validateConf(&conf)
	if err != nil {
		log.WithError(err).Fatal("Bad configuration")
	}
	server = reportToEmailServer{
		Conf: conf,
	}
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", server.Conf.GRPCport))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("server listening on %d", server.Conf.GRPCport)

	grpcServer := grpc.NewServer()
	proto.RegisterReportToEmailServer(grpcServer, &server)
	// determine whether to use TLS
	grpcServer.Serve(lis)
}

// SendEmail Handle gRPC request
func (server *reportToEmailServer) SendEmail(ctx context.Context, inEmail *proto.EmailToSend) (*proto.SentStatus, error) {

	// check auth token in request metadata
	mds, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		panic(ok)
	}
	tokenMd, ok := mds["token"]
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "token not found")
	}
	token := tokenMd[0]
	if _, ok := server.Conf.Tokens[token]; ! ok {
		return nil, status.Error(codes.PermissionDenied, "bad token")
	}

	// every auth token is bound with an office
	inEmail.Office = server.Conf.Tokens[token]
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
	file, err := os.Open(path.Join("emailTemplates", server.Conf.Email.TemplateName))
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
	e.From = server.Conf.Email.Sender
	e.To = []string{emailData.GetEmailAddress()}
	e.Subject = server.Conf.Email.Subject
	e.Text = []byte(emailBody)

	fileBuffer := bytes.NewBuffer(emailData.PdfPayload)
	e.Attach(fileBuffer, emailData.Filename, "application/pdf")

	auth := smtp.PlainAuth("", server.Conf.SMTP.Username, server.Conf.SMTP.Password, server.Conf.SMTP.Server)
	// TODO next line, get from config
	err := e.Send(fmt.Sprintf("%s:%s", server.Conf.SMTP.Server, server.Conf.SMTP.Port), auth)
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

	informat := "02012006"
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

func validateConf(conf *config) error {
	validate := validator.New()

	err := validate.Struct(conf)
	if err != nil {

		// check errors in validator
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.WithError(err).Error("Validator error")
			return err
		}

		// display struct validation errors
		for _, err := range err.(validator.ValidationErrors) {
			log.Errorf("ValidationError for field: %s, reason: %s\n", err.Namespace(), err.Tag())
		}
		return err
	}
	return nil
}
