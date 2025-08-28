CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE connections (
    user_one_id UUID NOT NULL,
    user_two_id UUID NOT NULL,
    
    -- ID của người đã gửi yêu cầu kết nối
    requester_id UUID NOT NULL,
    
    -- Trạng thái của mối quan hệ: 'pending' (chờ) hoặc 'accepted' (đã chấp nhận)
    status TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Thiết lập khóa ngoại (foreign key) để đảm bảo tính toàn vẹn dữ liệu
    FOREIGN KEY (user_one_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (user_two_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE CASCADE,

    -- Đặt khóa chính là cặp (user_one_id, user_two_id) để đảm bảo mỗi cặp chỉ có một mối quan hệ
    PRIMARY KEY (user_one_id, user_two_id),

    -- Ràng buộc (constraint) để đảm bảo dữ liệu luôn nhất quán
    CONSTRAINT check_status CHECK (status IN ('pending', 'accepted')),
    CONSTRAINT check_user_order CHECK (user_one_id::text < user_two_id::text),
    CONSTRAINT check_users_not_equal CHECK (user_one_id != user_two_id)
);

-- Tạo index để tăng tốc độ tìm kiếm các kết nối của một user
-- Index này sẽ giúp các câu lệnh WHERE (user_one_id = $1 OR user_two_id = $1) chạy nhanh hơn.
CREATE INDEX idx_connections_user_one ON connections (user_one_id);
CREATE INDEX idx_connections_user_two ON connections (user_two_id);