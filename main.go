package main

// The purpose of this program is to open a list of users
// and return all the repos the user has that are public via the
// github api.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Repo struct {
    FullName string `json:"full_name"`
}

func printRepos(body []byte) {
	// take the response body and parse for printRepos
	// TODO: filter out repose that were forked so we only scan repos we created

    var jsonData []Repo
    err := json.Unmarshal(body, &jsonData)
    if err != nil {
        log.Println("Error:", err)
    }
    for _, object := range jsonData {
        fmt.Println(object.FullName)
    }
}

func getToken() (string, error) {
	// load token file off disk
	// TODO: put token into a secure store

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Println(err)
	}
	file, err := os.Open(homedir + "/.github_token")
	if err != nil {
		log.Println("Unable to open token file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		return scanner.Text(), nil
	}

	return "", errors.New("empty token")
}

func makeRequest(url string, client *http.Client) *http.Response {
	// make a get request to github api to do something

	token, err := getToken()
	if err != nil {
		log.Println("Token error:", err)
		return nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error:", err)
	}
	req.Header.Add("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error", err)
	}

	return resp
}


var re := regexp.MustCompile(`\<(.*)\>\;\s+rel\=\"(.*)\"`)
func nextPage(link []string) (string, bool) {
	// github api does pagination so we need to handle that
	// https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api?apiVersion=2022-11-28

	linkMap := map[string]string{}
	for _, item := range strings.Split(link[0], ",") {
		match := re.FindStringSubmatch(item)
		linkMap[match[2]] = match[1]
	}

	if _, exists := linkMap["next"]; exists {
		return linkMap["next"], true
	}

	return "", false
}

func main() {
	url := "https://api.github.com/users/"
	client := &http.Client{Timeout: 5 * time.Second}

	file, err := os.Open("users.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		user := scanner.Text()
		resp := makeRequest(url + user + "/repos", client)
		defer resp.Body.Close()

		for {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("Error:", err)
			}

			if resp.StatusCode != 404 {
				printRepos(body)
			}

			var next string
			var doNext bool
			if len(resp.Header["Link"]) > 0 {
				next, doNext = nextPage(resp.Header["Link"])
			}

			if !doNext {
				break
			}

			resp = makeRequest(next, client)
		}

	}
}
