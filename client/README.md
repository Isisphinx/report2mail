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

## config

The program is looking for a YAML config file named `config` located in the same folder as the executable.  

``` yaml
cert: |
  -----BEGIN CERTIFICATE-----
  <x509 base64 encoded certificate authority>
  -----END CERTIFICATE-----
serverAddr: <IP>:<Port>

```

## usage

``` bash
# on *nix
./report2mail '{"emailAddress":"recipient@example.com","firstname":"Jean","lastname":"Test","date":"2019-12-08","office":"District Medical Imagery","fileLocation":"report.pdf"}'

# on Windows (in cmd.exe only)
# escaping double-quotes inside json payload is required
report2mail.exe "{\"emailAddress\":\"recipient@example.com\",\"firstname\":\"Jean\",\"lastname\":\"Test\",\"date\":\"2019-12-08\",\"office\":\"District Medical Imagery\",\"fileLocation\":\"report.pdf\"}"
```
