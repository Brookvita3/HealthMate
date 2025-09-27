package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

// Các hằng số để dễ dàng cấu hình
const (
	API_BASE_URL   = "https://api.fda.gov/drug/label.json"
	LIMIT_PER_PAGE = 100
	OUTPUT_DIR     = "openfda_data_concurrent"
	// Số lượng goroutine sẽ chạy đồng thời để tải dữ liệu
	NUM_WORKERS = 10
	// Lấy 1/10 tổng dữ liệu. Thay đổi thành 1 để lấy tất cả.
	DATA_FRACTION = 10
)

// Job định nghĩa một công việc: tải trang nào (PageNumber) với offset là bao nhiêu (Skip)
type Job struct {
	PageNumber int
	Skip       int
}

// MetaResponse dùng để lấy tổng số bản ghi từ API
type MetaResponse struct {
	Meta struct {
		Results struct {
			Total int `json:"total"`
		} `json:"results"`
	} `json:"meta"`
}

// worker là một goroutine sẽ nhận công việc từ channel `jobs` và thực hiện tải dữ liệu
func worker(id int, jobs <-chan Job, wg *sync.WaitGroup) {
	// Khi worker kết thúc, nó sẽ báo cho WaitGroup biết là đã xong
	defer wg.Done()

	// Lặp qua channel jobs. Vòng lặp sẽ tự kết thúc khi channel được đóng
	for job := range jobs {
		fmt.Printf("Worker %d đang xử lý trang %d...\n", id, job.PageNumber)

		apiURL := fmt.Sprintf("%s?limit=%d&skip=%d", API_BASE_URL, LIMIT_PER_PAGE, job.Skip)
		filename := fmt.Sprintf("%s/page_%d.json", OUTPUT_DIR, job.PageNumber)

		// Thử tải dữ liệu, nếu lỗi thì bỏ qua
		if err := fetchAndSave(apiURL, filename); err != nil {
			log.Printf("Lỗi ở Worker %d khi tải trang %d: %v", id, job.PageNumber, err)
		}
		// Có thể thêm một khoảng nghỉ nhỏ ở đây để tránh gây quá tải cho API
		// time.Sleep(100 * time.Millisecond)
	}
}

// fetchAndSave thực hiện logic tải và lưu file
func fetchAndSave(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API trả về mã lỗi: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, body, 0644)
}

func main() {
	fmt.Println("Bắt đầu quá trình tải dữ liệu (phiên bản đồng thời)...")

	// 1. Tạo thư mục output
	if err := os.MkdirAll(OUTPUT_DIR, os.ModePerm); err != nil {
		log.Fatalf("Không thể tạo thư mục '%s': %v", OUTPUT_DIR, err)
	}
	fmt.Printf("Dữ liệu sẽ được lưu trong thư mục: '%s'\n", OUTPUT_DIR)

	// 2. Lấy tổng số bản ghi một cách đồng bộ (chỉ 1 lần)
	var totalRecords int
	metaURL := fmt.Sprintf("%s?limit=1", API_BASE_URL)
	resp, err := http.Get(metaURL)
	if err != nil {
		log.Fatalf("Không thể lấy metadata từ API: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var meta MetaResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		log.Fatalf("Không thể phân tích metadata: %v", err)
	}
	totalRecords = meta.Meta.Results.Total

	// 3. Tính toán số lượng cần tải (1/10)
	recordsToFetch := totalRecords / DATA_FRACTION
	totalPagesToFetch := (recordsToFetch + LIMIT_PER_PAGE - 1) / LIMIT_PER_PAGE

	fmt.Printf("Tổng số bản ghi trên FDA: %d\n", totalRecords)
	fmt.Printf("Sẽ tải %d bản ghi (~1/%d) trong %d trang.\n", recordsToFetch, DATA_FRACTION, totalPagesToFetch)
	fmt.Printf("Sử dụng %d workers để tăng tốc.\n", NUM_WORKERS)

	// 4. Thiết lập worker pool
	jobs := make(chan Job, totalPagesToFetch)
	var wg sync.WaitGroup

	// Khởi tạo các workers
	for w := 1; w <= NUM_WORKERS; w++ {
		// Phải Add vào WaitGroup trước khi khởi chạy goroutine
		wg.Add(1)
		go worker(w, jobs, &wg)
	}

	// 5. Đẩy công việc vào channel jobs
	for i := 0; i < totalPagesToFetch; i++ {
		skip := i * LIMIT_PER_PAGE
		jobs <- Job{PageNumber: i + 1, Skip: skip}
	}
	// Đóng channel lại để báo cho các worker biết là không còn công việc nào nữa
	close(jobs)

	// 6. Đợi cho tất cả các worker hoàn thành
	wg.Wait()

	fmt.Println("\nHoàn tất! Đã tải xong phần dữ liệu thô được yêu cầu.")
}
