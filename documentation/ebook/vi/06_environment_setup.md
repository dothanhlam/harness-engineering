# Chương 6: Thiết lập Môi trường & Công cụ

Để chạy đường ống Kỹ thuật Harness thành công trên máy cục bộ của bạn, bạn cần cài đặt và cấu hình nền tảng công cụ cần thiết. Bộ điều phối (`main.go`) đóng vai trò là người nhạc trưởng, nhưng nó dựa vào các công cụ CLI bên ngoài này để thực hiện những công việc nặng nhọc.

## 1. Go (Golang)
Bộ điều phối và các module mã được tạo ra đều được viết bằng ngôn ngữ Go. Bạn sẽ cần **Go 1.21 trở lên**.
* **Mac/Linux**: 
  ```bash
  brew install go
  ```
* **Windows/Khác**: Tải xuống trình cài đặt từ [trang web chính thức của Go](https://go.dev/dl/).

## 2. Đặc vụ BA: Gemini CLI
Chúng ta sử dụng công cụ dòng lệnh `gemini` làm Nhà Phân tích Nghiệp vụ (Giai đoạn 0). Nó chịu trách nhiệm phân rã các tác vụ thô thành các danh sách kiểm tra nghiêm ngặt có trong tệp `definitions_of_done.md`.
* Đảm bảo bạn đã cài đặt và xác thực Gemini CLI trên máy của mình để các lệnh `gemini run` chạy liền mạch.

## 3. Đặc vụ Lập trình viên: Antigravity (`agy`)
Đặc vụ Lập trình viên (Giai đoạn 1) được hỗ trợ bởi CLI `agy` (Antigravity). Đây là một đặc vụ tự trị đọc các hệ thống gợi ý (system prompts) của chúng ta và viết mã vào bên trong thư mục `workspace/`.
* Cài đặt CLI `agy` theo các công cụ nội bộ của tổ chức bạn.
* Bộ điều phối tự động truyền cờ `--dangerously-skip-permissions` để cho phép `agy` chạy tự động mà không cần tạm dừng để xin quyền ghi tệp, tuy nhiên, nó vẫn bị giới hạn nghiêm ngặt (sandboxed) trong thư mục `workspace/` thông qua các cờ `--add-dir`.

## 4. Đặc vụ DevOps: Ollama
Đối với Giai đoạn 3 (Tạo Ghi chú Phát hành và Nén Bộ nhớ), chúng ta sử dụng các LLM cục bộ để tiết kiệm chi phí cho các API đám mây và đảm bảo sự riêng tư hoàn toàn cho mã nguồn của chúng ta.
* Tải xuống và cài đặt **Ollama** từ [ollama.com](https://ollama.com/).
* Khi đã cài đặt, hãy kéo (pull) mô hình mà chúng ta sử dụng để tạo tài liệu (được cấu hình trong `harness_config.json`):
  ```bash
  ollama pull hermes3:8b
  ```
* Hãy chắc chắn rằng máy chủ Ollama đang chạy ngầm trước khi bạn khởi động harness:
  ```bash
  ollama serve
  ```

## 5. Xác minh Cài đặt
Khi bạn đã cài đặt xong Go, Gemini, Agy, và Ollama, bạn có thể xác minh thiết lập của mình bằng cách chạy một bài kiểm tra đơn giản:
```bash
go run main.go -task "Tạo một module hello world đơn giản"
```

Nếu đường ống chạy thành công qua BA -> DEV -> QA -> AUDIT -> HITL -> DEVOPS mà không báo bất kỳ lỗi "command not found" nào, thì môi trường của bạn đã được định cấu hình hoàn hảo!
