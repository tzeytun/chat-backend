package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type     string `json:"type"`     
	Username string `json:"username"`
	Content  string `json:"content"`
	Time     string `json:"time"`    
}


var clients = make(map[*websocket.Conn]bool) // bağlı istemciler
var broadcast = make(chan Message)           // mesajları taşıyan kanal
var usernames = make(map[*websocket.Conn]string)     // bağlantı: kullanıcı adı eşleşmesi
var userListBroadcast = make(chan []string)          // kullanıcı listesi yayını
var typingBroadcast = make(chan string) // sadece yazan username'i taşır



var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		
		allowedOrigins := map[string]bool{
			"http://localhost:3000": true, // Geliştirme ortamı
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
			err := client.WriteJSON(struct {
				Type     string `json:"type"`
				Username string `json:"username"`
			}{
				Type:     "typing",
				Username: username,
			})
			if err != nil {
				log.Printf("Typing bildirimi hatası: %v", err)
				client.Close()
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
		loc = time.FixedZone("UTC+3", 3*60*60) // fallback
	}
	now := time.Now().In(loc)
	return now.Format("15:04")
}


// WebSocket bağlantılarını yöneten fonksiyon
func handleConnections(w http.ResponseWriter, r *http.Request) {


	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {

		username := usernames[ws]
		delete(clients, ws)
		delete(usernames, ws)
		broadcastUserList()

		broadcast <- Message{
		Type:     "system",
		Username: username,
		Content:  fmt.Sprintf("%s sohbetten ayrıldı", username),
		Time:     getCurrentTime(),
	}

		ws.Close()
	}()

	clients[ws] = true
	log.Println("Yeni istemci bağlandı!")

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

	// Eğer bu kullanıcı adı zaten kullanılıyorsa istemciye hata mesajı gönder
	for _, existingUsername := range usernames {
		if existingUsername == username {
			ws.WriteJSON(struct {
				Type    string `json:"type"`
				Content string `json:"content"`
			}{
				Type:    "error",
				Content: "Bu kullanıcı adı zaten kullanılıyor.",
			})
			ws.Close()
			return
		}
	}

	usernames[ws] = username
	broadcastUserList()
	continue // bu "join" mesajı broadcast edilmez
}

if msgType == "typing" {
	typingUsername := username
	typingBroadcast <- typingUsername
	continue
}


newMessage := Message{
	Type:     "message",
	Username: username,
	Content:  content,
	Time:     getCurrentTime(),
}
broadcast <- newMessage


	}
}

func handleUserListBroadcast() {
	for {
		userList := <-userListBroadcast
		log.Println("Güncellenmiş kullanıcı listesi:", userList) 
		for client := range clients {
			if client == nil {
				continue // 💡 null pointer koruması
			}
			err := client.WriteJSON(struct {
				Type string   `json:"type"`
				Users []string `json:"users"`
			}{
				Type:  "userlist",
				Users: userList,
			})
			if err != nil {
				log.Printf("Kullanıcı listesi gönderim hatası: %v", err)
				client.Close()
				delete(clients, client)
				delete(usernames, client)
			}
		}
	}
}


func broadcastUserList() {
	log.Println("Kullanıcı listesi güncellendi:", usernamesList)
	usernamesList := []string{}
	for _, name := range usernames {
		usernamesList = append(usernamesList, name)
	}
	userListBroadcast <- usernamesList
}


// Tüm istemcilere mesaj gönderen fonksiyon
func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Gönderim hatası: %v", err)
				client.Close()
				delete(clients, client)
				delete(usernames, client)
				broadcastUserList()
			}
		}
	}
}


func main() {
	http.HandleFunc("/ws", handleConnections)

	// PING endpoint'i: cold start önleme, uptime kontrolü
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	go handleMessages() // mesaj gönderici goroutine

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
