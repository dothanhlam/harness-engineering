# Chương 6: Thiết lập Môi trường & Công cụ

Để chạy pipeline Kỹ thuật Harness thành công trên máy cục bộ của bạn, bạn cần cài đặt và cấu hình nền tảng công cụ cần thiết. Orchestrator (`main.go`) đóng vai trò là người nhạc trưởng, nhưng nó dựa vào các công cụ CLI bên ngoài này để thực hiện những công việc nặng nhọc.

## 1. Go (Golang)
Orchestrator và các module mã được tạo ra đều được viết bằng ngôn ngữ Go. Bạn sẽ cần **Go 1.21 trở lên**.
* **Mac/Linux**: 
  ```bash
  brew install go
  ```
* **Windows/Khác**: Tải xuống trình cài đặt từ [trang web chính thức của Go](https://go.dev/dl/).

## 2. BA Agent: Gemini CLI
Chúng ta sử dụng công cụ dòng lệnh `gemini` làm Nhà Phân tích Nghiệp vụ (Giai đoạn 0). Nó chịu trách nhiệm phân rã các tác vụ thô thành các danh sách kiểm tra nghiêm ngặt có trong tệp `definitions_of_done.md`.
* Đảm bảo bạn đã cài đặt và xác thực Gemini CLI trên máy của mình để các lệnh `gemini run` chạy liền mạch.

## 3. Developer Agent: Antigravity (`agy`)
Developer Agent (Giai đoạn 1) được hỗ trợ bởi CLI `agy` (Antigravity). Đây là một agent tự trị đọc các hệ thống gợi ý (system prompts) của chúng ta và viết mã vào bên trong thư mục `workspace/`.
* Cài đặt CLI `agy` theo các công cụ nội bộ của tổ chức bạn.
* Orchestrator tự động truyền cờ `--dangerously-skip-permissions` để cho phép `agy` chạy tự động mà không cần tạm dừng để xin quyền ghi tệp, tuy nhiên, nó vẫn bị giới hạn nghiêm ngặt (sandboxed) trong thư mục `workspace/` thông qua các cờ `--add-dir`.

## 4. DevOps Agent: Ollama
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

## 5. Quản lý Hộp cát (Sandbox) & Kỹ năng (Skills)
Để giữ cho việc phát triển được tổ chức và duy trì các ngữ cảnh tác nhân mô-đun, pipeline hoạt động bên trong các thư mục cô lập (`workspace/`, `memory/`, `.agents/skills/`).

Bạn có thể khởi tạo các thư mục này và chuẩn bị bộ công cụ kỹ năng cục bộ của mình bằng cách sử dụng Makefile của Harness:

### Khởi tạo Hộp cát
Đầu tiên, chạy lệnh khởi tạo để tạo các thư mục cần thiết và khởi tạo tệp cấu hình cơ bản (baseline configuration):
```bash
make init
```

### Cung cấp Kỹ năng Tương tác
Tiếp theo, chạy trình cài đặt tương tác để chọn những kỹ năng tên miền chuyên gia từ danh mục awesome-skills để tải vào:
```bash
make skills
```
Kịch bản này sẽ kiểm tra Node và NPX, sau đó nhắc bạn chọn các gói chuyên gia (ví dụ: các mẫu thiết kế ClickHouse, hướng dẫn TDD, hoặc Kiến trúc Go Sạch).

Bạn cũng có thể liệt kê các kỹ năng đã cài đặt và loại bỏ chúng khi không còn cần thiết:
```bash
# Liệt kê tất cả các kỹ năng hiện đã cài đặt
make list-skills

# Loại bỏ một thư mục kỹ năng cụ thể
make remove-skill SKILL=<tên-thư-mục-kỹ-năng>
```

## 6. Xác minh Cài đặt
Khi bạn đã cấu hình xong môi trường và cung cấp các kỹ năng cần thiết, bạn có thể xác minh toàn bộ thiết lập:
```bash
make run -- -task "Tạo một module hello world đơn giản"
```

Nếu pipeline chạy thành công qua BA -> DEV -> QA (Audit & Tests) -> HITL -> DEVOPS -> MEMORY COMPACT mà không báo bất kỳ lỗi "command not found" nào, thì môi trường của bạn đã được định cấu hình hoàn hảo!
