package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/minhnbnt/uptime-monitor/generated/api"
)

const baseURL = "http://localhost:8080"

func authToken() string {

	reqBody := bytes.NewBufferString(`{"email":"seed@uptime.local","username":"seed","password":"seedseed","name":"Seed User"}`)

	ctx := context.Background()
	client := &http.Client{}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", reqBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("register: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		loginBody := bytes.NewBufferString(`{"login":"seed","password":"seedseed"}`)
		req, _ = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/login", loginBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			panic(fmt.Sprintf("login: %v", err))
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("auth: status %d\n  body: %s", resp.StatusCode, string(body)))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		panic(fmt.Sprintf("decode auth: %v", err))
	}

	return result.AccessToken
}

func main() {

	token := authToken()

	ports := make(chan int, 10000)

	go func() {
		for p := 10000; p <= 19999; p++ {
			ports <- p
		}
		close(ports)
	}()

	var (
		mu            sync.Mutex
		created       int
		createFails   int
		endpointSet   int
		endpointFails int
	)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {

			client := &http.Client{}

			for port := range ports {
				name := fmt.Sprintf("host.docker.internal:%d", port)

				reqBody, _ := json.Marshal(api.CreateServerRequest{Name: name})
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/api/v1/servers", bytes.NewReader(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)

				resp, err := client.Do(req)
				if err != nil {
					mu.Lock()
					createFails++
					mu.Unlock()
					fmt.Printf("FAIL create %s: %v\n", name, err)
					continue
				}

				if resp.StatusCode != http.StatusCreated {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					mu.Lock()
					createFails++
					mu.Unlock()
					fmt.Printf("FAIL create %s: status %d\n  body: %s\n", name, resp.StatusCode, string(body))
					continue
				}

				var srvResp api.ServerResponse
				if err := json.NewDecoder(resp.Body).Decode(&srvResp); err != nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					mu.Lock()
					createFails++
					mu.Unlock()
					fmt.Printf("FAIL decode %s: %v\n  body: %s\n", name, err, string(body))
					continue
				}
				resp.Body.Close()

				mu.Lock()
				created++
				mu.Unlock()

				parsedURL, _ := url.Parse(fmt.Sprintf("http://%s/health", name))
				epBody, _ := json.Marshal(&api.SetCheckMethodRequest{
					Method: api.CheckMethodTypePull,
					Endpoint: api.Endpoint{
						URL:          *parsedURL,
						Interval:     30,
						Timeout:      10,
						Method:       "GET",
						ExpectedCode: 200,
					},
				})

				req, _ = http.NewRequestWithContext(context.Background(), http.MethodPut,
					fmt.Sprintf(baseURL+"/api/v1/servers/%d/check_method", srvResp.Data.ID),
					bytes.NewReader(epBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)

				epResp, err := client.Do(req)
				if err != nil {
					mu.Lock()
					endpointFails++
					mu.Unlock()
					fmt.Printf("FAIL endpoint %s: %v\n", name, err)
					continue
				}

				if epResp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(epResp.Body)
					epResp.Body.Close()
					mu.Lock()
					endpointFails++
					fmt.Printf("FAIL endpoint %s: status %d\n  body: %s\n", name, epResp.StatusCode, string(body))
					mu.Unlock()
				} else {
					mu.Lock()
					endpointSet++
					mu.Unlock()
					fmt.Printf("DONE: %d\n", port)
					epResp.Body.Close()
				}
			}
		})
	}

	wg.Wait()

	fmt.Println("\n=== Result ===")
	fmt.Printf("Created:       %d\n", created)
	fmt.Printf("Create fails:  %d\n", createFails)
	fmt.Printf("Endpoint set:  %d\n", endpointSet)
	fmt.Printf("Endpoint fails:%d\n", endpointFails)
}
