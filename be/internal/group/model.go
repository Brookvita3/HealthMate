package group

// Định nghĩa các struct tương ứng với bảng trong DB
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ...
}

type GroupMember struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	// ...
}
