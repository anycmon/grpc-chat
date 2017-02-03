package server

import "sync"
import "sort"
import "github.com/anycmon/grpc-chat/chat"

type room struct {
	name  string
	id    string
	users roomUsers
}

type roomUsers []*chat.User

func (slice roomUsers) Len() int {
	return len(slice)
}

func (slice roomUsers) Less(i, j int) bool {
	return slice[i].Id < slice[j].Id
}

func (slice roomUsers) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (r *room) addUser(user *chat.User) {
	r.users = append(r.users, user)
	sort.Sort(r.users)
}

func (r *room) deleteUser(userId string) {
	i := sort.Search(len(r.users), func(n int) bool {
		return r.users[n].Id == userId
	})

	if i >= len(r.users) {
		return
	}

	r.users = append(r.users[:i], r.users[i+1:]...)
}

type rooms struct {
	collection []*room
	guard      sync.Mutex
}

func (rs *rooms) Find(id string) *room {
	rs.guard.Lock()
	defer rs.guard.Unlock()

	for _, r := range rs.collection {
		if r.id == id {
			return r
		}
	}

	return nil
}

func (rs *rooms) GetRooms() []chat.Room {
	result := []chat.Room{}
	rs.guard.Lock()
	defer rs.guard.Unlock()

	for _, r := range rs.collection {
		result = append(result, chat.Room{Id: r.id, Name: r.name})
	}

	return result
}

func (rs *rooms) GetRoomUsers(roomId string) []*chat.User {
	rs.guard.Lock()
	defer rs.guard.Unlock()
	users := []*chat.User{}

	for _, r := range rs.collection {
		if r.id == roomId {
			return r.users
		}
	}

	return users
}

func (rs *rooms) JoinRoom(room *chat.Room, user *chat.User) {
	rs.guard.Lock()
	defer rs.guard.Unlock()

	for _, r := range rs.collection {
		if r.id == room.Id {
			r.addUser(user)
		}
	}
}

func (rs *rooms) LeaveRoom(room *chat.Room, user *chat.User) {
	rs.guard.Lock()
	defer rs.guard.Unlock()

	for _, r := range rs.collection {
		if r.id == room.Id {
			r.deleteUser(user.Id)
		}
	}
}

func (rs *rooms) LeaveAllRooms(user *chat.User) {
	rs.guard.Lock()
	defer rs.guard.Unlock()

	for _, r := range rs.collection {
		r.deleteUser(user.Id)
	}
}
