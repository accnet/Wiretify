mạng (như WireGuard, iptables) mà còn là một Web Server mạnh mẽ.

Để tạo hệ thống navigation (điều hướng) chuyên nghiệp mà không phải lặp lại code ở mỗi trang, bạn nên dùng cơ chế **Template Layout**.

---

## 1. Cấu trúc thư mục đề xuất

Bạn nên tách riêng phần giao diện để dễ quản lý:

```text
/web
  /templates
    layout.html      # Chứa Header, Sidebar (Navigation), Footer
    dashboard.html   # Nội dung trang chủ
    port_forward.html # Nội dung trang Port Forwarding
    domains.html     # Nội dung trang quản lý Domain

```

## 2. Cách dùng Layout để làm Navigation

Thay vì copy thanh menu vào từng file, bạn định nghĩa một "khung" chung.

**File `templates/layout.html`:**

```html
<!DOCTYPE html>
<html>
<head><title>Gemini WG Panel</title></head>
<body>
    <nav>
        <a href="/">Dashboard</a> | 
        <a href="/ports">Port Forwarding</a> | 
        <a href="/domains">Domain Manager</a>
    </nav>
    <hr>
    
    {{template "content" .}}
    
    <footer>© 2026 My Go Panel</footer>
</body>
</html>

```

**File `templates/dashboard.html`:**

```html
{{define "content"}}
    <h1>Chào mừng bạn đến với Dashboard</h1>
    <p>Thông tin VPS: 137.184.85.113</p>
{{end}}

```

---

## 3. Code Go để điều hướng (Navigation)

Trong Go, bạn dùng gói `html/template` để "gắn" file nội bộ vào file khung.

```go
package main

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	// Gộp file layout và file trang cụ thể
	files := []string{
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", tmpl + ".html"),
	}
	
	t, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Thực thi và đẩy ra trình duyệt
	t.ExecuteTemplate(w, "layout.html", data)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "dashboard", nil)
	})

	http.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "port_forward", nil)
	})

	http.HandleFunc("/domains", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "domains", nil)
	})

	http.ListenAndServe(":8080", nil)
}

```

---

## 4. Ưu điểm của cách làm này

* **Dễ bảo trì:** Khi bạn muốn thêm một mục vào Menu, bạn chỉ cần sửa duy nhất file `layout.html`.
* **Hiệu suất:** Go compile template rất nhanh, tốn rất ít tài nguyên VPS.
* **Tương thích AlpineJS:** Vì bạn thích AlpineJS, bạn có thể nhúng thư viện này vào `layout.html` và nó sẽ hoạt động trên tất cả các trang con.

## 5. Mẹo nâng cao: Làm "Active Menu"

Để người dùng biết họ đang ở trang nào (ví dụ: bôi đậm mục "Port Forwarding"), bạn có thể truyền một biến `PageTitle` từ Go vào template:

```go
// Trong Go
renderTemplate(w, "port_forward", map[string]string{"CurrentPage": "ports"})

// Trong layout.html
<a href="/ports" class="{{if eq .CurrentPage "ports"}}active{{end}}">Port Forwarding</a>

```

Bạn có muốn tôi viết một bản **CSS Sidebar** đơn giản nhưng hiện đại để bạn dán vào file `layout.html` cho cái Panel của mình trông "xịn" như Tailscale không?