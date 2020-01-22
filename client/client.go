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
	_ "google.golang.org/grpc/credentials"
	"gopkg.in/go-playground/validator.v9"
)

type jsonInput struct {
	EmailAddress string `json:"emailaddress" validate:"required"`
	Lastname     string `json:"lastname" validate:"required"`
	Firstname    string `json:"firstname" validate:"required"`
	Date         string `json:"date" validate:"required"`
	Office       string `json:"office" validate:"required"`
	FileLocation string `json:"filelocation" validate:"required"`
}

func init() {
	// retrieve config from config file
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	// setup logger config
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	//log.SetReportCaller(true)
}

func main() {
	fmt.Println(os.Args)
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
	b := viper.GetString("cert") // CA cert
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM([]byte(b)) {
		panic("credentials: failed to append certificates")
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            cp,
	}

	conn, err := grpc.Dial(viper.GetString("serverAddr"), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.WithError(err).Fatal("Cannot reach server")
	}
	defer conn.Close()
	client := proto.NewReportToEmailClient(conn)

	jsonIn.FileLocation = filepath.Base(jsonIn.FileLocation)

	log.WithField("filename", jsonIn.FileLocation).Info("Ready to send mail")

	sentStatus, err := client.SendEmail(context.Background(), &proto.EmailToSend{
		EmailAddress: jsonIn.EmailAddress,
		Lastname:     jsonIn.Lastname,
		Firstname:    jsonIn.Firstname,
		Date:         jsonIn.Date,
		Office:       jsonIn.Office,
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
