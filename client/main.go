package main

import (
	"bufio"
	"flag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"grpc-chat/chat"
	"io"
	"os"
)

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
)

type ChatClient struct {
	user   *chat.User
	conn   *grpc.ClientConn
	client chat.ChatClient
}

func (c *ChatClient) Connect() {
	grpclog.Println("ChatClient::Connect: ...")
	var err error
	c.user, err = c.client.Connect(context.Background(), c.user)
	if err != nil {
		grpclog.Fatalf("ChatClient::Connect: %v", err)
	}

	grpclog.Printf("ChatClient::Connect: User connected as: %s, %s", c.user.Name, c.user.Id)

}

func (c *ChatClient) Rooms() chan *chat.Room {
	grpclog.Println("ChatClient::Rooms: ...")
	stream, err := c.client.Rooms(context.Background(), &chat.Empty{})
	if err != nil {
		grpclog.Fatalf("ChatClient::Rooms: %v", err)
	}

	rooms := make(chan *chat.Room)
	go func() {
		for true {
			room, err := stream.Recv()
			if err == io.EOF {
				close(rooms)
				break
			}

			if err != nil {
				grpclog.Fatalf("ChatClient::Rooms: %v", err)
				close(rooms)
				return
			}

			rooms <- room
		}
	}()

	return rooms
}

func (c *ChatClient) RoomUsers(room *chat.Room) chan *chat.User {
	grpclog.Println("ChatClient::RoomUsers: ...")
	stream, err := c.client.RoomUsers(context.Background(), room)
	if err != nil {
		grpclog.Fatalf("ChatClient::RoomUsers: %v", err)
	}

	users := make(chan *chat.User)
	go func() {
		for true {
			user, err := stream.Recv()
			if err == io.EOF {
				close(users)
				break
			}

			if err != nil {
				grpclog.Fatalf("ChatClient::RoomUsers: %v", err)
				close(users)
				return
			}

			users <- user
		}
	}()

	return users
}

func (c *ChatClient) JoinRoom(room *chat.Room) {
	grpclog.Println("ChatClient::JoinRoom: ...")
	c.client.JoinRoom(context.Background(), &chat.RoomCredentials{
		Room: room,
		User: c.user,
	})
}

func (c *ChatClient) LeaveRoom(room *chat.Room) {
	grpclog.Println("ChatClient::LeaveRoom: ...")
	c.client.LeaveRoom(context.Background(), &chat.RoomCredentials{
		Room: room,
		User: c.user,
	})
}

func (c *ChatClient) Close() {
	grpclog.Println("ChatClient::Close: ...")
	c.client.Disconnect(context.Background(), c.user)
	c.conn.Close()
}

func (c *ChatClient) ExchangeMsg(roomId string, msg <-chan *string, done <-chan bool) {
	stream, err := c.client.ExchangeMessages(context.Background())
	if err != nil {
		grpclog.Println(err)
		return
	}

	news := chat.News{
		SenderId: c.user.Id,
		RoomId:   roomId,
	}

	stream.Send(&news)

	go func() {
		for true {
			select {
			case msgToSend := <-msg:
				news := chat.News{
					SenderId: c.user.Id,
					RoomId:   roomId,
					Msg:      *msgToSend,
				}

				if err := stream.Send(&news); err != nil {
					grpclog.Println(err)
				}
			case <-done:
				return
			}
		}
	}()

	go func() {
		for true {
			select {
			case <-done:
				stream.CloseSend()
			default:
				if msg, err := stream.Recv(); err != nil {
					grpclog.Println(err)
				} else {
					grpclog.Printf("From room %s, user %s msg: %s", msg.RoomId, msg.SenderId, msg.Msg)
				}
			}
		}
	}()

}

func newChatClient(name string) *ChatClient {
	grpclog.Println("newChatClient: ...")
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		grpclog.Fatalf("newChatClient: fail to dial %v", err)
	}

	client := chat.NewChatClient(conn)
	chatClient := ChatClient{
		user:   &chat.User{Name: name},
		conn:   conn,
		client: client,
	}

	return &chatClient
}

func main() {
	client := newChatClient("Joe")
	defer client.Close()

	client.Connect()

	rooms := client.Rooms()

	roomsList := []*chat.Room{}
	for room := range rooms {
		grpclog.Printf("Room: %v", room)
		roomsList = append(roomsList, room)
	}

	roomUsers := client.RoomUsers(roomsList[0])

	for user := range roomUsers {
		grpclog.Printf("Room user: %v", user)
	}

	client.JoinRoom(roomsList[0])

	roomUsers = client.RoomUsers(roomsList[0])

	for user := range roomUsers {
		grpclog.Printf("Room user: %v", user)
	}

	//	client.LeaveRoom(roomsList[0])

	//	roomUsers = client.RoomUsers(roomsList[0])

	//	for user := range roomUsers {
	//		grpclog.Printf("Room user: %v", user)
	//	}

	sendChan := make(chan *string)
	doneChan := make(chan bool)

	client.ExchangeMsg(roomsList[0].Id, sendChan, doneChan)
	reader := bufio.NewReader(os.Stdin)
	for true {
		text, _ := reader.ReadString('\n')
		sendChan <- &text
	}
}
