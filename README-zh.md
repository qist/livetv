# LiveTV

将Youtube直播作为IPTV电视源

##安装方法

首先你需要安装Docker，Centos7用家可以直接使用参考这篇教学文档：[How To Install and Use Docker on CentOS 7]（https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-centos-7）

安装好Docker后，只需要使用以下命令即可在本地的9500连接埠启用LiveTV！

`docker run -d -p9500:9000 juestnow/livetv:main`

数据档存储于容器内的`/root/data`目录中，所以建议使用-v指令将这个目录映像到宿主机的目录。

一个使用外部储存目录的例子如下。

`docker run -d --name youtube --restart=always -p9500:9000 -v/mnt/data/livetv:/root/data juestnow/livetv:main`

这将在9500连接埠开启一个使用`/mnt/data/livetv`目录作为存储的LiveTV！容器。

PS:如果不指定外部存储目录，LiveTV！重新启动时将无法读取之前的设定档。

##使用方法

默认的登入密码是“password”，为了你的安全请及时修改。

首先你要知道如何在外界访问到你的主机，如果你使用VPS或者独立服务器，可以访问`http://你的主机ip:9500`，你应该可以看到以下画面：

![index_page](pic/index-zh.png)

首先你需要在设定区域点击“自动填充”，设定正确的URL。然后点击“储存设定”。

然后就可以添加频道，频道添加成功后就能M3U8档案列的地址进行播放了。

当你使用Kodi之类的播放器，可以考虑使用第一行的M3U档案URL进行播放，会自动生成包含所有频道信息的播放列表。

yt-dlp的文档可以在这里找到=> [https://github.com/yt-dlp/yt-dlp]（https://github.com/yt-dlp/yt-dlp）

nginx 代理设置

```nginx
upstream  youtube {
        least_conn;
        server 127.0.0.1:9000 max_fails=3 fail_timeout=30s resolve;
        keepalive 1000;
}

server {
    listen 80;
    server_name www.xxx.com;
     location / {
        proxy_pass http://youtube;
        proxy_redirect     off;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP    $remote_addr;
        proxy_set_header   X-Forwarded-For  $proxy_add_x_forwarded_for;
        proxy_next_upstream error timeout invalid_header http_502 http_503 http_504;
        proxy_max_temp_file_size 0;
        proxy_connect_timeout      90;
        proxy_send_timeout         90;
        proxy_read_timeout         90;
        proxy_buffer_size          4k;
        proxy_buffers              4 32k;
        proxy_busy_buffers_size    64k;
        proxy_temp_file_write_size 64k;
        proxy_http_version 1.1;
        proxy_set_header Accept-Encoding "";
   }
}
```