package main

import (
	"chat_Service/chat"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

type Connection struct {
	stream chat.ChatService_ConnectServer
	id     string
	active bool
	err    chan error
}

type Server struct {
	chat.UnimplementedChatServiceServer
	Connection []*Connection
	channel    map[string][]*chat.User
}

func main() {
	lis, err := net.Listen("tcp", "localhost:5400")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	chat.RegisterChatServiceServer(grpcServer, &Server{
		channel:    make(map[string][]*chat.User),
		Connection: []*Connection{},
	})
	fmt.Println(">>>> SERVER STARTED!")
	fmt.Println(">>>> Listening at port 5400")
	grpcServer.Serve(lis)
}

func (s *Server) Connect(user *chat.User, stream chat.ChatService_ConnectServer) error {
	conn := &Connection{
		stream: stream,
		id:     user.Username,
		active: true,
		err:    make(chan error),
	}
	s.Connection = append(s.Connection, conn)
	fmt.Printf("%s connected to server\n", user.Username)
	return <-conn.err
}

func (s *Server) CreateGroupChat(ctx context.Context, channel *chat.Channel) (*chat.Empty, error) {
	// check whether user connected to server
	conn, err := getUserConnection(s.Connection, channel.User.Username)
	if err != nil {
		return nil, err
	}

	// check whether user is active
	if !conn.active {
		return nil, userNotActiveError
	}

	// check whether the chanel is already exist
	_, ok := s.channel[channel.Name]
	if ok {
		return nil, channelAlreadyExistError
	}

	// creating new channel map and adding user to the channel
	s.channel[channel.Name] = []*chat.User{channel.User}
	fmt.Printf("new group with name %s is created\n", channel.Name)
	return &chat.Empty{}, nil
}

func (s *Server) JoinGroupChat(ctx context.Context, channel *chat.Channel) (*chat.Empty, error) {
	// check whether user connected to server
	conn, err := getUserConnection(s.Connection, channel.User.Username)
	if err != nil {
		return nil, err
	}

	// check whether user is active
	if !conn.active {
		return nil, userNotActiveError
	}

	// check whether the chanel is already exist
	_, ok := s.channel[channel.Name]
	if ok {
		// adding user to the channel
		s.channel[channel.Name] = append(s.channel[channel.Name], channel.User)
		return &chat.Empty{}, nil
	}
	fmt.Printf("%s joined to the channel %s\n", channel.User.Username, channel.Name)
	return nil, channelNotFoundError
}

func (s *Server) LeftGroupChat(ctx context.Context, channel *chat.Channel) (*chat.Empty, error) {
	// check whether user connected to server
	conn, err := getUserConnection(s.Connection, channel.User.Username)
	if err != nil {
		return nil, err
	}

	// check whether user is active
	if !conn.active {
		return nil, userNotActiveError
	}

	// check whether the chanel is already exist
	users, ok := s.channel[channel.Name]
	if ok {
		// removing user from the chat group
		userList := make([]*chat.User, 0)
		for _, user := range users {
			if user.Username != channel.User.Username {
				userList = append(userList, user)
			}
		}
		s.channel[channel.Name] = userList
	}
	fmt.Printf("%s left channel %s\n", channel.User.Username, channel.Name)
	return nil, channelNotFoundError
}

func (s *Server) ListChannels(ctx context.Context, currentUser *chat.User) (*chat.ChannelList, error) {
	channelList := make([]*chat.Channel, 0)
	for key, users := range s.channel {
		for _, user := range users {
			if currentUser.Username == user.Username {
				channelList = append(channelList, &chat.Channel{
					Name: key,
					User: user,
				})
			}
		}
	}
	return &chat.ChannelList{
		Channel: channelList,
	}, nil
}

func (s *Server) SendMessage(ctx context.Context, msg *chat.Message) (*chat.Empty, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	// check whether the message is a group message
	if msg.Channel != nil {
		fmt.Printf("%s send message to the group %s\n", msg.Sender.Username, msg.Channel.Name)
		// getting user list in the group
		users, ok := s.channel[msg.Channel.Name]
		if !ok {
			log.Println("channel not found")
			return nil, channelNotFoundError
		}
		for _, user := range users {
			// fetching connection object for each user
			conn, err := getUserConnection(s.Connection, user.Username)
			if err != nil || user.Username == msg.Sender.Username {
				// user connection not found in connection object. so we skip.
				log.Println(err)
				continue
			}
			wait.Add(1)

			go func(msg *chat.Message, conn *Connection) {
				defer wait.Done()

				// check whether the user is active
				if conn.active {
					err := conn.stream.Send(msg)
					log.Printf("Sending message %s to user %s\n", msg.Message, conn.id)

					if err != nil {
						log.Printf("Error with stream %v. Error: %v\n", conn.stream, err)
						conn.active = false
						conn.err <- err
					}
				}
			}(msg, conn)
		}
	} else {
		fmt.Printf("%s send message to user %s\n", msg.Sender.Username, msg.Receiver.Username)
		// fetching connection object for each user
		conn, err := getUserConnection(s.Connection, msg.Receiver.Username)
		if err != nil {
			// user connection not found in connection object
			log.Fatal(err)
			return nil, err
		}
		if conn.active {
			err := conn.stream.Send(msg)
			log.Printf("Sending message %s to user %s\n", msg.Message, conn.id)

			if err != nil {
				log.Printf("Error with stream %v. Error: %v\n", conn.stream, err)
				conn.active = false
				conn.err <- err
				return nil, err
			}
		}
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &chat.Empty{}, nil
}

func getUserConnection(conn []*Connection, username string) (*Connection, error) {
	for _, val := range conn {
		if val.id == username {
			return val, nil
		}
	}
	return nil, userNotFoundExistError
}
