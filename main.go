package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"strings"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Time     string `json:"time"`
}

type Client struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

func (c *Client) SafeWriteJSON(v interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.conn.WriteJSON(v)
}

var clients = make(map[*Client]bool)
var broadcast = make(chan Message)
var usernames = make(map[*Client]string)
var userListBroadcast = make(chan []string)
var typingBroadcast = make(chan string)
var lastMessageTimes = make(map[*Client]time.Time)


var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3000":                         true,
			"https://chat-frontend-kappa-nine.vercel.app": true,
		}
		if allowedOrigins[origin] {
			return true
		}
		log.Printf("Bloklanan origin: %s", origin)
		return false
	},
}

func handleTypingBroadcast() {
	for {
		username := <-typingBroadcast
		for client := range clients {
			err := client.SafeWriteJSON(struct {
				Type     string `json:"type"`
				Username string `json:"username"`
			}{
				Type:     "typing",
				Username: username,
			})
			if err != nil {
				log.Printf("Typing bildirimi hatası: %v", err)
				client.conn.Close()
				delete(clients, client)
				delete(usernames, client)
				broadcastUserList()
			}
		}
	}
}

func getCurrentTime() string {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.FixedZone("UTC+3", 3*60*60)
	}
	now := time.Now().In(loc)
	return now.Format("15:04")
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	client := &Client{conn: ws}

	clients[client] = true
	log.Println("Yeni istemci bağlandı!")

	defer func() {
		name, ok := usernames[client]
		if !ok {
			name = "Bilinmeyen"
		}
		delete(clients, client)
		delete(usernames, client)
		delete(lastMessageTimes, client) // 🧹 cooldown temizlik
		broadcastUserList()
		broadcast <- Message{
			Type:     "system",
			Username: name,
			Content:  fmt.Sprintf("%s sohbetten ayrıldı", name),
			Time:     getCurrentTime(),
		}
		client.conn.Close()
	}()

	for {
		var raw map[string]interface{}
		err := ws.ReadJSON(&raw)
		if err != nil {
			log.Printf("İstemci bağlantısı koptu: %v", err)
			break
		}

		msgType, _ := raw["type"].(string)
		username, _ := raw["username"].(string)
		content, _ := raw["content"].(string)

		if msgType == "join" {
			for _, existingUsername := range usernames {
				if strings.ToLower(existingUsername) == strings.ToLower(username) {
					client.SafeWriteJSON(struct {
						Type    string `json:"type"`
						Error   string `json:"error"`
						Content string `json:"content"`
					}{
						Type:    "error",
						Error:   "username_taken",
						Content: "Bu kullanıcı adı zaten kullanılıyor.",
					})
					client.conn.Close()
					return
				}
			}
			username = strings.ToLower(username)
			usernames[client] = username
			broadcastUserList()
			broadcast <- Message{
	Type:     "system",
	Username: username,
	Content:  fmt.Sprintf("%s sohbete katıldı", username),
	Time:     getCurrentTime(),
}
		}
			continue
		}

		if msgType == "typing" {
			typingBroadcast <- username
			continue
		}

		if msgType == "message" {
			now := time.Now()
			lastTime, exists := lastMessageTimes[client]
			if exists && now.Sub(lastTime) < time.Second {
				client.SafeWriteJSON(struct {
					Type    string `json:"type"`
					Error   string `json:"error"`
					Content string `json:"content"`
				}{
					Type:    "error",
					Error:   "cooldown",
					Content: "Lütfen yavaş yaz, spam algılandı.",
				})
				continue
			}
			lastMessageTimes[client] = now

			newMessage := Message{
				Type:     "message",
				Username: username,
				Content:  content,
				Time:     getCurrentTime(),
			}
			broadcast <- newMessage
		}
	}
}


func handleUserListBroadcast() {
	for {
		userList := <-userListBroadcast
		log.Println("Güncellenmiş kullanıcı listesi:", userList)
		for client := range clients {
			if client == nil {
				continue
			}
			err := client.SafeWriteJSON(struct {
				Type  string   `json:"type"`
				Users []string `json:"users"`
			}{
				Type:  "userlist",
				Users: userList,
			})
			if err != nil {
				log.Printf("Kullanıcı listesi gönderim hatası: %v", err)
				client.conn.Close()
				delete(clients, client)
				delete(usernames, client)
			}
		}
	}
}

func broadcastUserList() {
	usernamesList := []string{}
	for _, name := range usernames {
		usernamesList = append(usernamesList, name)
	}
	log.Println("Kullanıcı listesi güncelleniyor :", usernamesList)
	userListBroadcast <- usernamesList
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.SafeWriteJSON(msg)
			if err != nil {
				log.Printf("Gönderim hatası: %v", err)
				client.conn.Close()
				delete(clients, client)
				delete(usernames, client)
				broadcastUserList()
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	go handleMessages()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Panik engellendi:", r)
			}
		}()
		handleUserListBroadcast()
	}()
	go handleTypingBroadcast()

	fmt.Println("Sunucu 8080 portunda çoklu istemciyle dinliyor...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Sunucu başlatılamadı:", err)
	}
}