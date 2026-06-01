# Kỹ thuật Harness: Hướng dẫn dành cho Nhóm

Chào mừng bạn đến với Sách điện tử Kỹ thuật Harness! Khi tổ chức kỹ thuật của chúng ta mở rộng quy mô, chúng ta cần các hệ thống không chỉ giúp chúng ta viết mã mà còn thực sự tự động viết, kiểm tra và xác thực mã nguồn cùng với chúng ta.

Hướng dẫn ngắn gọn này được thiết kế để giúp toàn bộ nhóm kỹ thuật của chúng ta nắm bắt nhanh chóng về **Kỹ thuật Harness** và cách repository của chúng ta hoạt động.

## Kỹ thuật Harness là gì?

Kỹ thuật Harness nghe có vẻ đáng sợ, nhưng đối với bạn - một nhà phát triển, nó đơn giản là **đưa ra yêu cầu (prompting) cho orchestrator để nó làm những công việc nặng nhọc**. Thay vì phải gõ thủ công các đoạn mã lặp đi lặp lại, chạy các bài kiểm thử và viết tài liệu, bạn chỉ cần cung cấp cho Harness một yêu cầu và nó sẽ điều phối các AI agent (AI agents) để xây dựng phần mềm cho bạn.

Hãy nghĩ về nó như một dây chuyền lắp ráp tự động cho phần mềm của chúng ta:
1. Chúng ta cung cấp cho nó một **Yêu cầu Sản phẩm (PRD)**.
2. Harness giao nhiệm vụ cho các AI agent chuyên trách.
3. Harness kiểm tra khắt khe đầu ra, buộc AI phải tự sửa lỗi của chính nó.
4. Harness quét mã nguồn để tìm các lỗ hổng bảo mật.
5. Cuối cùng, nó đóng gói mã nguồn để con người đánh giá.

## Tại sao chúng ta lại áp dụng điều này?

* **Tốc độ chưa từng có**: Chúng ta có thể tạo ra toàn bộ các module và microservices chỉ trong vài phút.
* **Chất lượng được tích hợp sẵn**: Mã nguồn không chỉ được tạo ra; nó còn được biên dịch, kiểm tra và kiểm toán trước cả khi con người nhìn vào.
* **Tập trung vào Thiết kế Cấp cao**: Là những kỹ sư, công việc của chúng ta chuyển từ việc viết cú pháp sang thiết kế kiến trúc hệ thống và đưa ra các "Định nghĩa Hoàn thành" (Definitions of Done) khắt khe để AI tuân theo.

Trong các chương tiếp theo, chúng ta sẽ đi sâu vào chính xác cách repository cụ thể của chúng ta hoạt động, cấu trúc của pipeline, và cách chúng ta ngăn chặn các AI agent hoạt động ngoài tầm kiểm soát!
