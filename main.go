package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const uploadPath = "uploads"
const accessPassword = "123456"

type FileInfo struct {
	Name   string
	Path   string
	Folder string
}

//go:embed index.html
var content embed.FS

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./my_app <port>")
		fmt.Println("Usage: ./my_app 启动端口")
		return
	}

	port := os.Args[1]

	// 检查uploads文件夹是否存在，如果不存在则创建
	uploadsDir := "./uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err := os.Mkdir(uploadsDir, 0755)
		if err != nil {
			fmt.Printf("创建目录时出错: %v\n", err)
			return
		}
		fmt.Println("已创建目录。")
	}

	// 设置HTTP处理程序
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/files/", fileHandler)
	http.HandleFunc("/download/", downloadHandler)
	fmt.Printf("服务器已启动: http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// GET请求，显示上传文件列表页面或输入密码页面
		if r.URL.Path == "/" {
			handleMain(w, r)
		} else if r.URL.Path == "/login" {
			handleLogin(w, r)
		} else {
			http.NotFound(w, r)
		}
	} else if r.Method == http.MethodPost {
		// POST请求，处理文件上传逻辑，同时检查Cookie中的密码是否正确
		cookie, err := r.Cookie("access_password")
		if err != nil || cookie.Value != accessPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		uploadHandler(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	// 检查是否已经设置了访问密码的Cookie
	cookie, err := r.Cookie("access_password")
	if err != nil || cookie.Value != accessPassword {
		// 未设置密码Cookie或密码不匹配，重定向到输入密码页面
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 已设置且正确的密码Cookie，继续处理文件上传和显示页面的逻辑
	if r.Method == http.MethodPost {
		// 处理文件上传逻辑
		uploadHandler(w, r)
		return
	}

	// 显示上传文件列表页面
	files := listFiles(uploadPath)

	// 在传递给模板之前，修改每个文件的路径信息，添加上 uploadPath
	for i := range files {
		files[i].Path = filepath.Join(uploadPath, files[i].Path)
	}

	tmpl, err := template.ParseFS(content, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		file, handler, err := r.FormFile("uploadFile")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		folderName := r.FormValue("folderName")
		if folderName == "" {
			folderName = "." // 设置默认文件夹名称为当前目录 "."
		}
		folderPath := filepath.Join(uploadPath, folderName)
		err = os.MkdirAll(folderPath, os.ModePerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filePath := filepath.Join(folderPath, handler.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	files := listFiles(uploadPath)
	tmpl, err := template.ParseFS(content, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		// 验证访问密码
		cookie, err := r.Cookie("access_password")
		if err != nil || cookie.Value != accessPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		filePath := strings.TrimPrefix(r.URL.Path, "/files/")
		fullPath := filepath.Join(filePath)

		// 检查filePath是否以uploadPath开头
		if !strings.HasPrefix(fullPath, uploadPath) {
			http.Error(w, "Invalid file path", http.StatusBadRequest)
			return
		}

		err = os.Remove(fullPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/download/")
	fullPath := filepath.Join(filePath)

	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 设置响应头，告知浏览器以附件形式下载文件
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))
	w.Header().Set("Content-Type", "application/octet-stream")

	// 将文件内容写入响应主体
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func listFiles(root string) []FileInfo {
	var files []FileInfo
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			relativePath := strings.TrimPrefix(path, root+string(os.PathSeparator))
			folderName := filepath.Dir(relativePath)
			// 获取文件所在的具体文件夹名称
			_, folder := filepath.Split(folderName)
			if folder == "" {
				folder = filepath.Base(root) // 根目录的文件夹名使用根目录名称
			}
			files = append(files, FileInfo{Name: info.Name(), Path: relativePath, Folder: folder})
		}
		return nil
	})
	if err != nil {
		fmt.Println("读取文件时出错:", err)
	}
	return files
}

func init() {
	http.HandleFunc("/login", handleLogin)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		password := r.FormValue("password")
		if password == accessPassword {
			// 设置密码Cookie，有效期设置为7天
			cookie := http.Cookie{
				Name:     "access_password",
				Value:    password,
				MaxAge:   7 * 24 * 60 * 60, // 7天
				HttpOnly: true,
			}
			http.SetCookie(w, &cookie)
			// 登录成功后重定向回主页面
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			// 密码错误，显示错误信息或重新登录页面
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// 显示输入密码的表单页面
	fmt.Fprintln(w, `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Login</title>
        <style>
            body {
                font-family: Arial, sans-serif;
                background-color: #f0f0f0;
                margin: 0;
                padding: 0;
                display: flex;
                justify-content: center;
                align-items: center;
                height: 100vh;
            }

            .container {
                width: 300px;
                background-color: #fff;
                padding: 20px;
                border-radius: 5px;
                box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            }

            h1 {
                text-align: center;
                margin-bottom: 20px;
            }

            form {
                display: flex;
                flex-direction: column;
            }

            input[type=password] {
                padding: 10px;
                margin-bottom: 10px;
                border: 1px solid #ccc;
                border-radius: 3px;
                font-size: 16px;
            }

            button[type=submit] {
                padding: 10px 20px;
                background-color: #007bff;
                color: #fff;
                border: none;
                border-radius: 3px;
                font-size: 16px;
                cursor: pointer;
            }

            button[type=submit]:hover {
                background-color: #0056b3;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>Login</h1>
            <form method="post">
                <input type="password" name="password" placeholder="Enter access password" required>
                <button type="submit">Login</button>
            </form>
        </div>
    </body>
    </html>
`)
}
