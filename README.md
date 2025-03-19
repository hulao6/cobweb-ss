# 简介

互联网访问工具分享管理后台

# 为什么做这个

我想教人编程, 但编程的工具链都需要翻墙, 所以我做了这个分享网络给好友, 以便他们进行开发

# 使用

```sh
docker run -d --name cobweb --restart always -v $PWD/pb_data/:/app/pb_data/ -p 10000:10000 shynome/cobweb:v3.0.0
# 密码和邮箱记得改成自己喜欢的
docker exec -ti cobweb /app/cobweb superuser create admin@cobweb.www adminadmin
```

管理界面: http://127.0.0.1:10000/_/

# 开发测试

```sh
git clone -b v3 https://github.com/shynome/cobweb.git
# 启动服务端
go run . serve --http 127.0.0.1:10000
# 另起终端启动客户端
v2ray run -c client.jsonc
# 另起终端测试客户端是否连接成功
# 获取你的地址
curl -x socks5://127.0.0.1:1080 myip.ipip.net
# 禁止访问内网地址
curl -x socks5://127.0.0.1:1080 127.0.0.1
```
