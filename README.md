# 编译前的操作

**更改访问密码**

```
`const accessPassword = "123456"`
```

**更改存储目录和自动创建目录**

```
`const uploadPath = "./uploads"`
```

```
`uploadsDir := "./uploads"`
```

# 编译

```
go build -o my_app
```

# 交叉编译

```
export GOOS=linux GOARCH=amd64; go build -o my_app
```

# 使用

```
./my_app 8080
```

****访问127.0.0.1:8080****
