package domain

type Record struct {
	Hash         string `json:"hash"`
	AttemptsLeft int    `json:"attempts_left"`
	CreatedAt    int64  `json:"created_at"`
}
