# Chương 5: Cấu trúc Mã nguồn & Các Module

Để làm việc hiệu quả với repository Kỹ thuật Harness, điều quan trọng là phải hiểu cách tổ chức mã nguồn. Repository được chia thành các thành phần cấu trúc dùng để cung cấp năng lượng cho AI và các gói chức năng mà AI tạo ra.

## Các Tệp Cốt lõi

### `main.go`
Đây là trái tim của orchestrator. Nó quản lý cỗ máy trạng thái (state machine), ủy quyền các tác vụ cho các agent, thực thi các rào chắn bảo mật (`AuditGeneratedCode`), và nắm bắt các dữ liệu từ xa (telemetry).

Ví dụ, trạng thái của pipeline được theo dõi một cách nghiêm ngặt bằng cách sử dụng struct `WorkflowState` này, nó sẽ được ghi vào tệp `state.json` ở mỗi lần chuyển trạng thái:
```go
// WorkflowState là trạng thái đường ống được lưu giữ, ghi vào workspace/state.json.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	RetryCount   int       `json:"retry_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

Nó cũng bao gồm logic kiểm toán bảo mật nghiêm ngặt của chúng ta để ngăn chặn AI làm bất cứ điều gì có tính chất phá hoại:
```go
if strings.Contains(code, "rm -rf") {
	auditErr = fmt.Errorf("tệp %s chứa lệnh terminal phá hoại 'rm -rf'", path)
	return fmt.Errorf("lỗi kiểm toán")
}
```

### `harness_config.json`
Tệp cấu hình toàn cục. Nó quyết định việc sử dụng các đặc vụ nào (ví dụ: `gemini`, `agy`, `ollama`), thiết lập số lần tự sửa lỗi tối đa, và xác định các điểm cuối (endpoints) API.

## Môi trường của AI

* **`.agents/`**: Chứa các hệ thống gợi ý (system prompts) và cấu hình hành vi cho các AI agent của chúng ta. Ví dụ, `antigravity_dev_prompt.md` hướng dẫn Developer agent chính xác về các tiêu chuẩn viết mã của chúng ta.
* **`memory/`**: Đây là "bộ não" của AI. Không giống như các kỹ sư con người, các AI agent sẽ mất ngữ cảnh của chúng giữa các lần chạy. Chúng ta lưu giữ bộ nhớ của chúng tại đây:
    * `definitions_of_done.md`: Danh sách kiểm tra kỹ thuật nghiêm ngặt do BA agent tạo ra.
    * `mem0-server/`: Backend cơ sở dữ liệu vector Mem0 cục bộ được sử dụng để tìm kiếm ngữ nghĩa.
    * `lessons_learned.md`: Một tài liệu sống, nơi AI ghi lại những lỗi mà nó đã sửa để không lặp lại chúng nữa.

## Không gian Làm việc được Tạo ra (Workspace)

* **`workspace/`**: Đây là không gian thử nghiệm (sandbox) nơi mọi phép thuật xảy ra. Tất cả mã do AI tạo ra đều được đặt ở đây. Hiện tại, AI của chúng ta đã tạo thành công và xác thực một số module có độ bảo mật cao:
    * `password/`: Một thư viện băm (hashing) bcrypt mạnh mẽ được tối ưu hóa cho các kỹ thuật phân bổ bộ nhớ bằng không (zero-allocation) hiệu suất cao.
    * `email_validation/`: Một module có các bài kiểm thử đơn vị (unit-tested) nghiêm ngặt dành cho việc xác thực email.
    * `landing_page/`: Một trang web tiếp thị hoàn chỉnh, độc lập và mang phong cách glassmorphic.
    * `random/`: Một tiện ích tạo ngẫu nhiên an toàn.
    * `fibonacci/`: Một trình tạo chuỗi Fibonacci có độ chính xác tùy ý và được tối ưu hóa cao.
    * `state.json`: Tệp theo dõi lưu giữ giai đoạn thực thi hiện tại của đường ống.
    * `telemetry.json`: Dữ liệu phân tích (payload) được tạo ra ở cuối mỗi chu kỳ thực thi thành công. Đây là những gì chúng ta theo dõi:
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

Bằng cách giữ cho orchestrator (`main.go`) tách biệt hoàn toàn khỏi mã được tạo ra (`workspace/`), chúng ta đảm bảo rằng AI của mình có thể xây dựng phần mềm dạng module, có thể kiểm thử mà không bao giờ phá vỡ chính pipeline cốt lõi.
