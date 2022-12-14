# ChatApp

ChatApp is a live chat application written in go. ChatApp make use of gRPC stream and protobuf 3.0 libraries.

# Usage

1. run server
```
go run server.go
```

2. run client
```
go run client.go -user test -server :8000
```
This will connect the user ```test``` to the server

## Commands
Once the user connected to server, you can use different pre-defined commands to execute different operations

The available operations are List channel, Create channel, Join Channel, Leave channel, Send Direct message, send message to channels

1. List Channels joined by the user.
```
list
```
2. Create new channel
```
create>channel_name
```
3. Join a channel
```
join>channel_name
```
4. Leave a channel
```
exit>channel_name
```
5. Send direct message to other user
```
send>user>message
```
6. Send Message in a channel user joined
```
broadcast>channel_name>message
```

# Examples
```
list

create>Crypto_channel

join>Bitcoin_channel

exit>Crypto_channel

send>John>Hi John

broadcast>Bitcoin_channel>good morning
```