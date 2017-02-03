package server

import (
	"github.com/anycmon/grpc-chat/chat"
	"github.com/satori/go.uuid"
	"google.golang.org/grpc/grpclog"
	"sync"
)

type user struct {
	name        string
	id          string
	isListening bool
	sendBuffer  chan *chat.News
	bufferGuard sync.Mutex
}

func (u *user) Send(news *chat.News) {
	grpclog.Println("User::Send ...")
	u.bufferGuard.Lock()
	defer u.bufferGuard.Unlock()
	if u.isListening {
		u.sendBuffer <- news
	}
	grpclog.Println("User::Send")
}

func (u *user) OpenSendStream(stream chat.Chat_ExchangeMessagesServer) {
	u.bufferGuard.Lock()
	defer u.bufferGuard.Unlock()

	if u.isListening == false {
		u.sendBuffer = make(chan *chat.News, 10)
		u.isListening = true
		go func() {
			for true {
				if err := stream.Send(<-u.sendBuffer); err != nil {
					grpclog.Println(err)
					u.CloseStream()
					break
				}
			}

		}()
	}
}

func (u *user) CloseStream() {
	u.bufferGuard.Lock()
	defer u.bufferGuard.Unlock()

	if u.isListening == true {
		u.isListening = false
		close(u.sendBuffer)
	}
}

type users struct {
	collection []*user
	guard      sync.Mutex
}

func (us *users) add(u *chat.User) {
	newUser := user{
		name: u.Name,
		id:   GenerateUID(),
	}

	us.guard.Lock()
	defer us.guard.Unlock()

	us.collection = append(us.collection, &newUser)
	u.Id = newUser.id
}

func (us *users) remove(id string) {
	us.guard.Lock()
	defer us.guard.Unlock()

	for i, u := range us.collection {
		if u.id == id {
			us.collection = append(us.collection[:i], us.collection[i+1:]...)
		}
	}
}

func (us *users) pop(id string) *user {
	us.guard.Lock()
	defer us.guard.Unlock()

	for i, user := range us.collection {
		if user.id == id {
			us.collection = append(us.collection[:i], us.collection[i+1:]...)
			return user
		}
	}

	return nil
}

func (us *users) find(id string) *user {
	us.guard.Lock()
	defer us.guard.Unlock()

	for _, u := range us.collection {
		if u.id == id {
			return u
		}
	}

	return nil
}

func GenerateUID() string {
	return uuid.NewV4().String()
}
