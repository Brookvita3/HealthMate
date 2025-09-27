package realtime

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

// Hàm helper để tạo một client giả (không có kết nối thật)
func newFakeClient(m *Manager, userId string) *Client {
	return &Client{
		manager: m,
		conn:    &websocket.Conn{},
		userId:  userId,
		send:    make(chan *OutgoingMessage, 256),
	}
}

func BenchmarkBroadcastToNListeners(b *testing.B) {
	// Các kịch bản khác nhau: 10, 100, 1000 người nghe
	benchmarks := []struct {
		name      string
		listeners int
	}{
		{"10-Listeners", 10},
		{"100-Listeners", 100},
		{"1000-Listeners", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// --- SETUP ---
			// Tạo manager và sender
			manager := NewManager(nil) // Không cần service thật
			sender := newFakeClient(manager, "sender-id")

			// Tạo N listeners và subscribe họ vào kênh của sender
			for i := 0; i < bm.listeners; i++ {
				listener := newFakeClient(manager, "listener-id")
				// Kiểm tra và khởi tạo map nếu chưa có
				if manager.subscriptions[sender.userId] == nil {
					manager.subscriptions[sender.userId] = make(map[*Client]bool)
				}
				manager.subscriptions[sender.userId][listener] = true
				// Chạy một goroutine để đọc từ kênh send, nếu không broadcast sẽ bị block
				go func() {
					for range listener.send {
					}
				}()
			}

			// Tin nhắn giả để broadcast
			msg := &IncomingMessage{Action: ActionWearableDataUpdate, Payload: json.RawMessage(`{"hr":75}`)}

			// Reset timer để không tính thời gian setup
			b.ResetTimer()

			// --- RUN ---
			// Chạy vòng lặp benchmark
			for i := 0; i < b.N; i++ {
				manager.handleBroadcast(sender, msg)
			}
		})
	}
}
