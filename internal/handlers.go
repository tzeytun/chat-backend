package internal

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	
	"github.com/gorilla/websocket"
)

var Clients = make(map[*Client]bool)
var Broadcast = make(chan Message)
var Usernames = make(map[*Client]string)
var UserListBroadcast = make(chan []string)
var TypingBroadcast = make(chan string)
var LastMessageTimes = make(map[*Client]time.Time)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3000": true,
			"https://chat-frontend-kappa-nine.vercel.app": true,
		}
		if allowedOrigins[origin] {
			return true
		}
		log.Printf("Bloklanan origin: %s", origin)
		return false
	},
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade hatası:", err)
		return
	}
	client := &Client{Conn: ws}
	Clients[client] = true
	log.Println("Yeni istemci bağlandı!")

	defer func() {
		name := Usernames[client]
		delete(Clients, client)
		delete(Usernames, client)
		delete(LastMessageTimes, client)
		BroadcastUserList()
		Broadcast <- Message{
			Type:     "system",
			Username: name,
			Content:  fmt.Sprintf("%s sohbetten ayrıldı", name),
			Time:     getCurrentTime(),
		}
		client.Conn.Close()
	}()

	for {
		var raw map[string]interface{}
		err := ws.ReadJSON(&raw)
		if err != nil {
			log.Printf("Bağlantı koptu: %v", err)
			break
		}

		msgType, _ := raw["type"].(string)
		content, _ := raw["content"].(string)
		username := Usernames[client]

		if username == "" && msgType != "join" {
			client.SafeWriteJSON(map[string]string{
				"type":    "error",
				"error":   "unauthorized",
				"content": "Unauthorized!",
			})
			client.Conn.Close()
			return
		}

		switch msgType {
		case "join":
			joinUsername, _ := raw["username"].(string)
			joinUsername = strings.TrimSpace(strings.ToLower(joinUsername))
			if joinUsername == "" || len(joinUsername) > 20 || !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(joinUsername) {
				client.SafeWriteJSON(map[string]string{
					"type":    "error",
					"error":   "invalid_username",
					"content": "Geçersiz kullanıcı adı. Sadece harf ve rakam içermeli.",
				})
				client.Conn.Close()
				return
			}
			for _, name := range Usernames {
				if name == joinUsername {
					client.SafeWriteJSON(map[string]string{
						"type":    "error",
						"error":   "username_taken",
						"content": "Bu kullanıcı adı zaten kullanılıyor.",
					})
					client.Conn.Close()
					return
				}
			}
			Usernames[client] = joinUsername
			BroadcastUserList()
			Broadcast <- Message{
				Type:     "system",
				Username: joinUsername,
				Content:  fmt.Sprintf("%s sohbete katıldı", joinUsername),
				Time:     getCurrentTime(),
			}

		case "typing":
			TypingBroadcast <- username

		case "message":
			if content == "" || len(content) > 500 {
				client.SafeWriteJSON(map[string]string{
					"type":    "error",
					"error":   "invalid_message",
					"content": "Geçersiz mesaj.",
				})
				continue
			}
			now := time.Now()
			lastTime := LastMessageTimes[client]
			if now.Sub(lastTime) < time.Second {
				client.SafeWriteJSON(map[string]string{
					"type":    "error",
					"error":   "cooldown",
					"content": "Lütfen yavaş yaz.",
				})
				continue
			}
			LastMessageTimes[client] = now
			Broadcast <- Message{
				Type:     "message",
				Username: username,
				Content:  content,
				Time:     getCurrentTime(),
			}
		}
	}
}

func HandleMessages() {
	for msg := range Broadcast {
		for client := range Clients {
			if err := client.SafeWriteJSON(msg); err != nil {
				log.Println("Gönderim hatası:", err)
				client.Conn.Close()
				delete(Clients, client)
				delete(Usernames, client)
				BroadcastUserList()
			}
		}
	}
}

func HandleUserListBroadcast() {
	for users := range UserListBroadcast {
		for client := range Clients {
			err := client.SafeWriteJSON(struct {
				Type  string   `json:"type"`
				Users []string `json:"users"`
			}{"userlist", users})
			if err != nil {
				client.Conn.Close()
				delete(Clients, client)
				delete(Usernames, client)
			}
		}
	}
}

func BroadcastUserList() {
	list := []string{}
	for _, name := range Usernames {
		list = append(list, name)
	}
	UserListBroadcast <- list
}

func HandleTypingBroadcast() {
	for username := range TypingBroadcast {
		for client := range Clients {
			err := client.SafeWriteJSON(struct {
				Type     string `json:"type"`
				Username string `json:"username"`
			}{"typing", username})
			if err != nil {
				client.Conn.Close()
				delete(Clients, client)
				delete(Usernames, client)
				BroadcastUserList()
			}
		}
	}
}

func getCurrentTime() string {
	loc, _ := time.LoadLocation("Europe/Istanbul")
	return time.Now().In(loc).Format("15:04")
}
