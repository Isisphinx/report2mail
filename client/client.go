package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/isisphinx/report2mail/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"gopkg.in/go-playground/validator.v9"
)

type jsonInput struct {
	EmailAddress string `json:"emailaddress" validate:"required"`
	Lastname     string `json:"lastname" validate:"required"`
	Firstname    string `json:"firstname" validate:"required"`
	Date         string `json:"date" validate:"required"`
	FileLocation string `json:"filelocation" validate:"required"`
}

func init() {
	// retrieve config from config file
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.BindEnv("serverAddr")
	viper.BindEnv("token")
	err := viper.ReadInConfig()
	switch err.(type) {
	case nil:
		break
	case viper.ConfigFileNotFoundError:
		log.Debug("No config file found.")
	default:
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	if viper.Get("serverAddr") == nil {
		log.Fatal("no serverAddr in conf or env")
	}

	// setup logger config
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	//log.SetReportCaller(true)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Program arguments")
		for i := 0; i < len(os.Args); i += 1 {
			fmt.Printf("%d: %s\n", i, os.Args[i])
		}
		panic(fmt.Sprintf("Wrong arguments: want %d, got %d", 2, len(os.Args)))
	}

	// retrieve and validate user data from command line arguments
	jsonIn, err := parseAndValidate([]byte(os.Args[1]))
	if err != nil {
		log.WithError(err).Fatal("Invalid input")
	}

	// read payload file into buffer
	fileContent, err := ioutil.ReadFile(jsonIn.FileLocation)
	if err != nil {
		log.WithError(err).Fatal("failed to read file")
	}

	// prepare connection and client
	var creds credentials.TransportCredentials
	sysCerts, err := x509.SystemCertPool()
	if err != nil {
		log.Errorf("failed to load system certificates: %v. Fallback to skipVerify", err)
		config := &tls.Config{
			InsecureSkipVerify: true,
		}
		creds = credentials.NewTLS(config)
	} else {
		creds = credentials.NewClientTLSFromCert(sysCerts, "")
	}

	conn, err := grpc.Dial(viper.GetString("serverAddr"), grpc.WithTransportCredentials(creds))
	if err != nil {
		log.WithError(err).Fatal("Cannot reach server")
	}
	defer conn.Close()
	client := proto.NewReportToEmailClient(conn)

	jsonIn.FileLocation = filepath.Base(jsonIn.FileLocation)

	// build header and add token
	md := metadata.New(map[string]string{"token": viper.GetString("token")})
	header := metadata.NewOutgoingContext(context.Background(), md)

	log.WithField("filename", jsonIn.FileLocation).Info("Ready to send mail")

	sentStatus, err := client.SendEmail(header, &proto.EmailToSend{
		EmailAddress: jsonIn.EmailAddress,
		Lastname:     jsonIn.Lastname,
		Firstname:    jsonIn.Firstname,
		Date:         jsonIn.Date,
		Filename:     jsonIn.FileLocation,
		PdfPayload:   fileContent,
	})
	if err != nil {
		log.WithError(err).Error("Failed to send email")
		os.Exit(1)
	}
	log.WithField("filename", jsonIn.FileLocation).Infof("Mail status on server: %#v", sentStatus.Status)
}

func parseAndValidate(in []byte) (*jsonInput, error) {
	dec := json.NewDecoder(bytes.NewBuffer(in))
	dec.DisallowUnknownFields()
	jsonIn := new(jsonInput)
	err := dec.Decode(jsonIn)
	if err != nil {
		log.WithError(err).Fatal("cannot parse json input")
	}

	validate := validator.New()

	err = validate.Struct(jsonIn)
	if err != nil {

		// check errors in validator
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.WithError(err).Error("Validator error")
			return nil, err
		}

		// display struct validation errors
		for _, err := range err.(validator.ValidationErrors) {
			log.Errorf("ValidationError for field: %s, reason: %s\n", err.Namespace(), err.Tag())
		}
		return nil, err
	}
	return jsonIn, nil
}
