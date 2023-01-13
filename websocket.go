package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type Event struct {
	event    string `json:e`
	resource string `json:r`
	state    State
}

type State struct {
	lastupdated string `json:lastupdated`
	open        bool   `json:open`
}

func main() {

	godotenv.Load(".env")
	var addressString = fmt.Sprintf("%s:%s", os.Getenv("IP_ADDRESS"), os.Getenv("PORT"))
	var addr = flag.String("addr", addressString, "http service address")
	var gotifyPath = fmt.Sprintf("https://gotify.azaurus.dev/message?token=%s", os.Getenv("GOTIFY_API_KEY"))
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			s := string(message)
			if strings.Contains(s, "\"open\":true") {
				fmt.Println("Door Open detected")
				postData := url.Values{
					"title":    {"Door opened"},
					"message":  {"The front door is open"},
					"priority": {"5"},
				}
				resp, err := http.PostForm(gotifyPath, postData)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(resp)
			} else if strings.Contains(s, "\"vibration\":true") {
				fmt.Println("Vibration detected")
				postData := url.Values{
					"title":    {"Vibration detected- websocket"},
					"message":  {"There is vibration at the front door"},
					"priority": {"5"},
				}
				resp, err := http.PostForm(gotifyPath, postData)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(resp)
			}
			//log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
