package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
)

func main() {
	url := "http://localhost:8080/user/data"

	dataTemplate := `{
        "user_id": "55b3d3af-d7e8-4f45-9bc6-e87140a030b0",
        "type": "heart_rate",
        "value": %d,
        "timestamp": "2025-11-06T08:30:00Z"
    }`

	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			jsonData := []byte(fmt.Sprintf(dataTemplate, v))
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			defer resp.Body.Close()
			fmt.Println("Status:", resp.Status)
		}(i)
	}

	wg.Wait()
	fmt.Println("Finished sending 100 requests")
}
