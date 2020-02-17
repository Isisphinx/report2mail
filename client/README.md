# GRPC client for isisphinx/report2mail

## compilation


``` bash
# for *nix
go build -o report2mail
```

This application is intended to run on Windows. Use the following line to cross compile on linux for windows target.

``` bash
# for Windows
GOOS=windows GOARCH=amd64 go build -o report2mail.exe
```

## usage

``` bash
# on *nix
SERVERADDR="127.0.0.1:51000" TOKEN="AuthToken" ./report2mail '{"emailAddress":"recipient@example.com","firstname":"Jean","lastname":"Test","date":"2019-12-08","office":"District Medical Imagery","fileLocation":"report.pdf"}'

# on Windows (in cmd.exe only)
set SERVERADDR=127.0.0.1:51000
set TOKEN=authToken
# escaping double-quotes inside json payload is required
report2mail.exe "{\"emailAddress\":\"recipient@example.com\",\"firstname\":\"Jean\",\"lastname\":\"Test\",\"date\":\"2019-12-08\",\"office\":\"District Medical Imagery\",\"fileLocation\":\"report.pdf\"}"
```
