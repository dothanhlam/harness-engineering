# Chương 5: Cấu trúc Mã nguồn & Các Module

Để làm việc hiệu quả với repository Kỹ thuật Harness, điều quan trọng là phải hiểu cách tổ chức mã nguồn. Repository được chia thành bộ điều phối (orchestrator) dạng module dùng để cung cấp năng lượng cho AI và các gói chức năng mà AI tạo ra.

## Lõi của Orchestrator

### `main.go`
Một điểm khởi đầu (entrypoint) gọn nhẹ. Nó phân tích các lệnh con `init` / `run` và các cờ của chúng, tải cấu hình, áp dụng các ghi đè từ CLI, và trao quyền xử lý cho các gói pipeline. Phần việc nặng nhọc nằm trong `internal/`.

### Các gói `internal/`
Orchestrator đã được tái cấu trúc từ một tệp nguyên khối (monolithic) duy nhất thành các gói tập trung:

* **`internal/pipeline/`** — các vòng lặp cốt lõi và cỗ máy trạng thái.
    * `core.go`: vòng lặp tuần tự BA→Dev→QA→HITL→DevOps.
    * `epic.go`: Trình điều phối Epic (phân rã một thư mục các yêu cầu, chạy các tác vụ con tuần tự hoặc song song).
    * `stages.go`: các hằng số `Stage` của pipeline, `MaxRetries`, và `WorkflowState` được lưu giữ vào `workspace/state.json`.
    * `helpers.go`: xây dựng prompt và phân tích tệp được tạo ra.
* **`internal/agent/`** — bộ chuyển đổi (adapter) agent có thể cắm thêm. Thực thi một binary `llama_cpp` cục bộ hoặc một CLI agent dạng mẫu (ví dụ Claude) và phân tích mức sử dụng token vào telemetry.
* **`internal/config/`** — tải `harness_config.json`, hợp nhất nó lên trên các mặc định tích hợp sẵn.
* **`internal/qa/`** — **kiểm toán bảo mật** đồng thời (`AuditGeneratedCode`) và **trình chạy kiểm thử** (tự động phát hiện Go / Node / Python).
* **`internal/memory/`** — cập nhật và nén `memory/system_blueprint.md`.
* **`internal/telemetry/`** — trình theo dõi các chỉ số thực thi được bảo vệ bằng mutex.

Trạng thái của pipeline được theo dõi một cách nghiêm ngặt bằng cách sử dụng struct `WorkflowState` (`internal/pipeline/stages.go`), nó sẽ được ghi vào `workspace/state.json` ở mỗi lần chuyển giai đoạn:
```go
// WorkflowState is the persisted pipeline state written to workspace/state.json.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	RetryCount   int       `json:"retry_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

Logic kiểm toán bảo mật ngăn chặn AI làm bất cứ điều gì có tính chất phá hoại nằm trong `internal/qa`, được điều khiển bởi bộ quy tắc có thể cấu hình:
```go
// Each rule maps a forbidden substring to the audit failure it triggers.
if strings.Contains(code, pattern) {
	return fmt.Errorf("file %s %s", path, reason)
}
```

### `harness_config.json`
Tệp cấu hình toàn cục. Nó quyết định việc sử dụng agent và model nào theo từng giai đoạn (mặc định tất cả đều là `llama_cpp`), cửa sổ ngữ cảnh (context window), flash-attention, danh sách `qa_ignore`, và các ghi đè quy tắc QA tùy chọn.

## Môi trường của AI

* **`.agents/`**: Chứa các hệ thống gợi ý (system prompts) cho các AI agent của chúng ta. Ví dụ, `antigravity_dev_prompt.md` hướng dẫn Developer agent phát ra các khối `SEARCH/REPLACE` chính xác để nó vá mã hiện có thay vì ghi đè lên nó.
* **`memory/`**: Đây là "bộ não" của AI. Không giống như các kỹ sư con người, các AI agent sẽ mất ngữ cảnh của chúng giữa các lần chạy. Chúng ta lưu giữ bộ nhớ của chúng tại đây:
    * `definitions_of_done.md`: Danh sách kiểm tra kỹ thuật nghiêm ngặt do BA agent tạo ra.
    * `system_blueprint.md`: Một bản đồ kiến trúc giúp AI hiểu về hệ thống rộng lớn hơn mà nó đang đóng góp vào (được cập nhật và nén bởi `internal/memory`).
    * `lessons_learned.md`: Một tài liệu sống chứa các hướng dẫn gỡ lỗi và lịch sử vận hành.

## Không gian Làm việc được Tạo ra (Workspace)

* **`workspace/`**: Đây là hộp cát (sandbox) nơi mọi mã do AI tạo ra được đặt vào. Mỗi tính năng có một thư mục con sạch sẽ riêng — ví dụ `password/` (băm bcrypt), `email_validation/`, `landing_page/`, `random/`, và `fibonacci/`. Nó cũng chứa `state.json` (giai đoạn pipeline trực tiếp) và `RELEASE_NOTES.md`.
* **`telemetry.json`** (được ghi vào thư mục gốc của dự án ở cuối mỗi lần chạy thành công):
      ```go
      type Telemetry struct {
          TotalDurationSeconds float64  `json:"total_duration_seconds"`
          StagesExecuted       []string `json:"stages_executed"`
          TotalRetriesUsed     int      `json:"total_retries_used"`
          CodeHealingSuccess   bool     `json:"code_healing_success"`
          LinesOfCodeGenerated int      `json:"lines_of_code_generated"`
          Timestamp            string   `json:"timestamp"`
      }
      ```

Bằng cách giữ cho orchestrator (`main.go` + `internal/`) tách biệt hoàn toàn khỏi mã được tạo ra (`workspace/`), chúng ta đảm bảo rằng AI của mình có thể xây dựng phần mềm dạng module, có thể kiểm thử mà không bao giờ phá vỡ chính pipeline cốt lõi.
