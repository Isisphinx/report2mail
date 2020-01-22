module github.com/Isisphinx/report2mail/client

go 1.13

require (
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/isisphinx/report2mail/proto v0.0.0-00010101000000-000000000000
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/viper v1.6.1
	google.golang.org/grpc v1.21.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.30.2
)

replace github.com/isisphinx/report2mail/proto => ../proto
