# Simple Syrup

### An SSH honeypot in Go
This is a simple little SSH honeypot I wrote. It advertises an SSH server with no security, performs a handshake with anything that connects to it, and then logs anything sent through the connection.

Also, there is a little desktop notification program that pings you whenever something connects to the honeypot.

Hopefully, some bots will connect to it and direct me to the cool malware :)

## How To Use
```
ssh-keygen -t rsa
go run sshpot.go
```

## TODO
- Make control flow more understandable
- Create a fake terminal prompt to send through the connection
- Make logs more understandable
