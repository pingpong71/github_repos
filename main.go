package main

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
)

func print_repos(body []byte) {
// take the response body and parse for print_repos
// TODO: filter out repose that were forked so we only scan repos we created

    var json_data []map[string]interface{}
    err := json.Unmarshal(body, &json_data)
    if err != nil {
        log.Println("Error:", err)
    }

    for _, object := range json_data {
        fmt.Println(object["full_name"])
    }
}

func has_repos(body []byte) bool {
// make sure the user we are checking on has repos
// if they do not github returns a message like 
// {"message": "Not Found"}

    var message map[string]interface{}
    err := json.Unmarshal(body, &message)
    if err != nil {
       return true 
    }

    return false
}

func get_token() (string, error) {
// load token file off disk
// TODO: put token into a secure store

    homedir, err := os.UserHomeDir()
    if err != nil {
        log.Fatal(err)
    }
    file, err := os.Open(homedir + "/.github_token")
    if err != nil {
        log.Fatal("Unable to open token file")
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        return scanner.Text(), nil
    }

    return "", errors.New("empty token") 
}

func make_request(url string) *http.Response {
// make a get request to github api to do something
    
    token, err := get_token()
    if err != nil {
        log.Fatal("Token error:", err)
    }
    req, err := http.NewRequest("GET", url, nil)
    req.Header.Add("Authorization", token)
    client := &http.Client{}

    resp, err := client.Do(req)
    if err != nil {
        log.Println("Error", err)
    }

    return resp
}

func next_page(link []string) (string, bool) {
// github api does pagination so we need to handle that
// https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api?apiVersion=2022-11-28

    link_map := map[string]string{}
    for _, item := range strings.Split(link[0], ",") {
        re := regexp.MustCompile(`\<(.*)\>\;\s+rel\=\"(.*)\"`)
        match := re.FindStringSubmatch(item)
        link_map[match[2]] = match[1]
    }
    
    if _, exists := link_map["next"]; exists {
        return link_map["next"], true
    }

    return "", false
}

func main() {
    url := "https://api.github.com/users/"

    file, err := os.Open("users.txt")
    if err != nil {
        log.Fatal(err)
    }

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        user := scanner.Text()
        resp := make_request(url + user + "/repos")
        defer resp.Body.Close()
        for {
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                log.Println("Error:", err)
            }

            if has_repos(body) {
                print_repos(body)
            }

            var next string
            var do_next bool
            if len(resp.Header["Link"]) > 0 {
                next, do_next = next_page(resp.Header["Link"])
            }

            if !do_next {
                break
            }

            resp = make_request(next)
        }

    }
}




