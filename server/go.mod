module github.com/isisphinx/report2mail/server

go 1.13

require (
	github.com/isisphinx/report2mail/proto v0.0.0-00010101000000-000000000000
	github.com/jordan-wright/email v0.0.0-20190819015918-041e0cec78b0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.6.1
	google.golang.org/grpc v1.25.1
)

replace github.com/isisphinx/report2mail/proto => ../proto
