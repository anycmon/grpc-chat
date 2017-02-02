package server

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"
	"grpc-chat/chat"
	"io"
)

type ChatServer struct {
	rooms rooms
	users users
}

func New() *ChatServer {
	return &ChatServer{
		rooms: rooms{
			collection: []*room{
				&room{name: "Foo", id: "1"},
			},
		},
	}
}

func (s *ChatServer) Disconnect(c context.Context, user *chat.User) (*chat.Empty, error) {
	grpclog.Printf("ChatServer::Disconnect: %s ...", user.Name)
	s.rooms.LeaveAllRooms(user)
	serverUser := s.users.pop(user.Id)
	if serverUser != nil {
		serverUser.CloseStream()
	}

	grpclog.Printf("ChatServer::Disconnect: User disconnected %s, %s", user.Name, user.Id)
	return &chat.Empty{}, nil
}

func (s *ChatServer) Connect(c context.Context, u *chat.User) (*chat.User, error) {
	grpclog.Printf("ChatServer::Connect: %s ...", u.Name)
	if user := s.users.find(u.Id); user != nil {
		errMsg := "User already connected"
		grpclog.Printf("ChatServer::Connect: %s", errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	s.users.add(u)

	grpclog.Printf("ChatServer::Connect: User connected %s, %s", u.Name, u.Id)
	return u, nil
}

func (s *ChatServer) Rooms(empty *chat.Empty, stream chat.Chat_RoomsServer) error {
	grpclog.Println("ChatServer::Rooms: ...")
	rooms := s.rooms.GetRooms()

	for _, room := range rooms {
		if err := stream.Send(&room); err != nil {
			grpclog.Println("ChatServer::Rooms: %s", err.Error())
			return err
		}
	}

	grpclog.Println("ChatServer::Rooms: done")
	return nil
}

func (s *ChatServer) RoomUsers(room *chat.Room, stream chat.Chat_RoomUsersServer) error {
	grpclog.Printf("ChatServer::RoomUsers: %s", room.Id)
	users := s.rooms.GetRoomUsers(room.Id)

	for _, user := range users {
		if err := stream.Send(user); err != nil {
			grpclog.Println("ChatServer::Rooms: %s", err.Error())
			return err
		}
	}

	grpclog.Printf("ChatServer::RoomUsers: Done")
	return nil
}

func (s *ChatServer) JoinRoom(c context.Context, credentials *chat.RoomCredentials) (*chat.Empty, error) {
	grpclog.Printf("ChatServer::JoinRoom: room %s, user %s",
		credentials.Room.Id, credentials.User.Id)
	s.rooms.JoinRoom(credentials.Room, credentials.User)

	grpclog.Printf("ChatServer::JoinRoom: done")
	return &chat.Empty{}, nil
}

func (s *ChatServer) LeaveRoom(c context.Context, credentials *chat.RoomCredentials) (*chat.Empty, error) {
	grpclog.Printf("ChatServer::LeaveRoom: room %s, user %s",
		credentials.Room.Id, credentials.User.Id)
	s.rooms.LeaveRoom(credentials.Room, credentials.User)

	grpclog.Printf("ChatServer::LeaveRoom: done")
	return &chat.Empty{}, nil
}

func (s *ChatServer) ExchangeMessages(stream chat.Chat_ExchangeMessagesServer) error {
	//GetRoomUsers
	//Iterate over it and push data to user send channel

	news, err := stream.Recv()

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	if user := s.users.find(news.SenderId); user == nil {
		//ERROR handling

		return nil
	} else {
		user.OpenSendStream(stream)
	}

	return s.recvMsgLoop(stream)
}

func (s *ChatServer) recvMsgLoop(stream chat.Chat_ExchangeMessagesServer) error {
	for true {
		news, err := stream.Recv()

		if err == io.EOF {

			break
		}

		if err != nil {
			return err
		}

		roomUsers := s.rooms.GetRoomUsers(news.GetRoomId())
		grpclog.Println(news)
		for _, roomUser := range roomUsers {

			if user := s.users.find(roomUser.Id); user != nil {
				user.Send(news)
			}
		}
	}

	return nil
}
