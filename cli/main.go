package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const signalBaseURL = "http://localhost:8080"

func registerToken(token string) (*http.Response, error) {
	resp, err := http.Post(fmt.Sprintf("%s/register?token=%s", signalBaseURL, token), "", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		log.Println(bodyString)
	}
	return resp, err
}

func main() {
	mode := flag.String("mode", "S", "S for send, R for receive. Default is S")
	//sourcePath := flag.String("src", "/home/mahadevan/test.txt", "Path of the file to send")
	token := flag.String("token", "", "Token which the sender and receiver must know (Required)")

	flag.Parse()

	if *token == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	switch *mode {
	case "S":
		registerToken(*token)
	case "R":
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}
