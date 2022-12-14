package main

import (
	"bufio"
	"chat_Service/chat"
	"context"
	"strings"

	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"sync"
)

var username = flag.String("user", "default", "username")
var tcpServer = flag.String("server", ":5000", "Tcp server")

const (
	createChannelOperation    = "create"
	joinChannelOperation      = "join"
	leaveChannelOperation     = "exit"
	sendMessageOperation      = "send"
	broadcastMessageOperation = "broadcast"
	listChannelOperation      = "list"
)

func main() {
	flag.Parse()

	fmt.Print("--- CLIENT APP ---")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

	conn, err := grpc.Dial(*tcpServer, opts...)
	if err != nil {
		log.Fatalf("\nFail to dail: %v", err)
	}

	defer conn.Close()

	ctx := context.Background()
	client := chat.NewChatServiceClient(conn)
	wait := sync.WaitGroup{}
	connect(ctx, client, wait)
}

func connect(ctx context.Context, client chat.ChatServiceClient, wait sync.WaitGroup) error {
	var streamError error
	done := make(chan struct{})

	fmt.Printf("\nconnecting user %s to server >>\n>>", *username)
	stream, err := client.Connect(ctx, &chat.User{
		Username: *username,
	})

	if err != nil {
		return fmt.Errorf("connect failed: %v\n>>", err)
	}

	wait.Add(1)

	go func(str chat.ChatService_ConnectClient) {
		defer wait.Done()
		for {
			msg, err := str.Recv()
			if err != nil {
				streamError = fmt.Errorf("ERROR:reading message>> %v\n>>", err)
				break
			}
			if msg.Channel != nil {
				// print group message as [group-name] username: message
				fmt.Printf("[%v] %v: %s\n>>", msg.Channel.Name, msg.Sender.Username, msg.Message)
			} else {
				// print direct message as username: message
				fmt.Printf("%v: %s\n>>", msg.Sender.Username, msg.Message)
			}
		}
	}(stream)

	wait.Add(1)
	go func() {
		defer wait.Done()
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			command := strings.Split(scanner.Text(), ">")
			// check for list channel operation
			if strings.TrimSpace(command[0]) == listChannelOperation {
				listChannels(ctx, client)
				// check for command of format operation>channelName
			} else if len(command) == 2 {
				// check whether operation is to create channel; create>channelName
				if strings.TrimSpace(command[0]) == createChannelOperation {
					createGroup(ctx, client, command[1])
					//  check whether operation is to join channel; join>channelName
				} else if strings.TrimSpace(command[0]) == joinChannelOperation {
					joinGroup(ctx, client, command[1])
					//  check whether operation is to leave channel; exit>channelName
				} else if strings.TrimSpace(command[0]) == leaveChannelOperation {
					leaveGroup(ctx, client, command[1])
					//  check whether operation is to send direct message; send>Message
				} else {
					fmt.Println("Invalid command\n>>")
				}
				//  check whether operation is to send message
			} else if len(command) == 3 {
				// send>user>message
				if strings.TrimSpace(command[0]) == sendMessageOperation {
					sendMessage(ctx, client, command[1], command[2], nil)
					// broadcast>channel>message
				} else if strings.TrimSpace(command[0]) == broadcastMessageOperation {
					sendMessage(ctx,
						client,
						command[1],
						command[2],
						&chat.Channel{Name: strings.TrimSpace(command[1])})
				} else {
					fmt.Println("Invalid command\n>>")
				}
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done

	return streamError
}

func createGroup(ctx context.Context, client chat.ChatServiceClient, channelName string) {
	c := chat.Channel{
		Name: strings.TrimSpace(channelName),
		User: &chat.User{Username: strings.TrimSpace(*username)},
	}
	_, err := client.CreateGroupChat(ctx, &c)
	if err != nil {
		fmt.Printf("ERROR:creating group>> %v\n>>", err)
	}
	fmt.Print("\n>>")
}

func joinGroup(ctx context.Context, client chat.ChatServiceClient, channelName string) {
	c := chat.Channel{
		Name: strings.TrimSpace(channelName),
		User: &chat.User{Username: strings.TrimSpace(*username)},
	}
	_, err := client.JoinGroupChat(ctx, &c)
	if err != nil {
		fmt.Printf("ERROR:joining group>> %v\n>>", err)
	}
	fmt.Print("\n>>")
}

func leaveGroup(ctx context.Context, client chat.ChatServiceClient, channelName string) {
	c := chat.Channel{
		Name: strings.TrimSpace(channelName),
		User: &chat.User{Username: strings.TrimSpace(*username)},
	}
	_, err := client.LeftGroupChat(ctx, &c)
	if err != nil {
		fmt.Printf("ERROR:left group>> %v\n>>", err)
	}
	fmt.Print("\n>>")
}

func sendMessage(ctx context.Context, client chat.ChatServiceClient, receiver, message string, channel *chat.Channel) {
	msg := &chat.Message{
		Receiver: &chat.User{Username: strings.TrimSpace(receiver)},
		Sender:   &chat.User{Username: strings.TrimSpace(*username)},
		Message:  message,
		Channel:  channel,
	}
	_, err := client.SendMessage(ctx, msg)
	if err != nil {
		fmt.Printf("ERROR:sending message>> %v\n>>", err)
	}
	fmt.Print("\n>>")
}

func listChannels(ctx context.Context, client chat.ChatServiceClient) {
	channelList, err := client.ListChannels(ctx, &chat.User{Username: strings.TrimSpace(*username)})
	if err != nil {
		fmt.Printf("ERROR:listing channels>> %v\n>>", err)
	}
	for _, list := range channelList.Channel {
		fmt.Printf("%s/%s", list.User.Username, list.Name)
	}
	fmt.Print("\n>>")
}
