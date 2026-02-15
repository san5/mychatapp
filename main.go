package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
	User string `json:"user"`
	Text string `json:"text"`
	Time string `json:"time"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]string)
var mutex sync.Mutex
var broadcast = make(chan Message)

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// username first
	_, nameMsg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}
	username := string(nameMsg)

	mutex.Lock()
	if len(clients) >= 2 {
		mutex.Unlock()
		conn.WriteMessage(websocket.TextMessage,
			[]byte(`{"type":"system","text":"Room full"}`))
		conn.Close()
		return
	}
	clients[conn] = username
	mutex.Unlock()

	broadcast <- Message{
		Type: "system",
		Text: username + " joined",
		Time: now(),
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			removeClient(conn, username)
			return
		}

		broadcast <- Message{
			Type: "msg",
			User: username,
			Text: string(msg),
			Time: now(),
		}
	}
}

func removeClient(conn *websocket.Conn, username string) {
	mutex.Lock()
	delete(clients, conn)
	mutex.Unlock()
	conn.Close()

	broadcast <- Message{
		Type: "system",
		Text: username + " left",
		Time: now(),
	}
}

func handleBroadcast() {
	for {
		msg := <-broadcast
		data, _ := json.Marshal(msg)

		mutex.Lock()
		for c := range clients {
			c.WriteMessage(websocket.TextMessage, data)
		}
		mutex.Unlock()
	}
}

func now() string {
	return time.Now().Format("15:04")
}

func main() {
	http.HandleFunc("/ws", handleWS)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	go handleBroadcast()
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
