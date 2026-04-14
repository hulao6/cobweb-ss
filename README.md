# 简介

互联网访问工具分享管理后台

# 为什么做这个

我想教人编程, 但编程的工具链都需要翻墙, 所以我做了这个分享网络给好友, 以便他们进行开发

# claw cloud 使用教程(视频)

趁现在还有热度, 放个 claw cloud 的 aff 注册链接: https://console.run.claw.cloud/signin?link=TVY0NDGJPJWR

注: 由于 claw cloud 错误地删除了我的数据, 不再进行推荐 (教程还放在这是觉得视频可以作个参考)

https://github.com/user-attachments/assets/767cf8e4-6881-4235-9931-339d54946ff6

# 使用

```sh
docker run -d --name cobweb --restart always -v $PWD/pb_data/:/app/pb_data/ -p 10000:10000 shynome/cobweb:v3.3.0
# 现在默认会创建下方的用户记得删除或修改密码 (无需自行创建了)
docker exec -ti cobweb /app/cobweb superuser create admin@cobweb.example admin@cobweb.example
```

管理界面: http://127.0.0.1:10000/_/

注意: [v2rayA 的 trojan 配置不支持 websocket](https://github.com/v2rayA/v2rayA/discussions/1790)

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
