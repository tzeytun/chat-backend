package main

import (
	"fmt"
	"log"
	"net/http"

	"chat-backend/internal"
)

func main() {
	http.HandleFunc("/ws", internal.HandleConnections)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	go internal.HandleMessages()
	go internal.HandleUserListBroadcast()
	go internal.HandleTypingBroadcast()

	fmt.Println("Sunucu 8080 portunda çoklu istemciyle dinliyor...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Sunucu başlatılamadı:", err)
	}
}
