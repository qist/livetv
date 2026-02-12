# LiveTV

[中文文档](README-zh.md)
Use Youtube live as IPTV feeds

## Install 

First you need to install Docker, Centos7 users can directly use the following tutorials. [How To Install and Use Docker on CentOS 7](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-centos-7)

After installing Docker, you can enable LiveTV! on your local port 9000 with the following command:

`docker run -d -p9000:9000 juestnow/livetv:latest`

`ghcr.io/qist/livetv:latest`

The data file is stored inside the container in the `/root/data` directory, so it is recommended to use the -v command to map this directory to the host's directory.

An example of using an external storage directory is shown below.

`docker run -d --name youtube --restart=always -p9000:9000  -v/mnt/data/livetv:/root/data juestnow/livetv:latest`

` docker run -d --name youtube --restart=always --net=host  -v/opt/data/livetv:/root/data ghcr.io/qist/livetv:latest`

This will open a LiveTV! container on port 9000 that uses the `/mnt/data/livetv` directory as storage.

PS: If you do not specify an external storage directory, LiveTV! will not be able to read the previous configuration file when it reboots.

## Usage

Default password is "password".

First you need to know how to access your host from the outside, if you are using a VPS or a dedicated server, you can visit `http://your_ip:9000` and you should see the following screen.

![index_page](pic/index-en.png)

First of all, you need to click "Auto Fill" in the setting area, set the correct URL, then click "Save Config".

Then you can add a channel. After the channel is added successfully, you can play the M3U8 file from the address column.

When you use Kodi or similar player, you can consider using the M3U file URL in the first row to play, and a playlist containing all the channel information will be generated automatically.

yt-dlp document here => [https://github.com/yt-dlp/yt-dlp](https://github.com/yt-dlp/yt-dlp)

## Troubleshooting Pull Failures

If the pull URL fails, try the following in order (from simple to more advanced):

1. First, append runtime and component parameters:

```bash
--js-runtimes deno --remote-components ejs:npm -f b -g {url}
```

Notes:
- `--js-runtimes deno --remote-components ejs:npm` enables JS-based extractors (required for some YouTube pages).
- This parameter set also works for VOD parsing.

2. If it still fails, add the player client and User-Agent:

```bash
--js-runtimes deno --remote-components ejs:npm --extractor-args 'youtube:player_client=web' --add-header "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36" -f b -g {url}
--js-runtimes deno --remote-components ejs:npm --extractor-args 'youtube:player-client=android' -f b -g {url}
```

## Cookies: How to Obtain and Configure

Some channels require login. You can provide cookies so yt-dlp can access them.

### How to get cookies.txt (Netscape format)
1. Sign in to the target website in your browser (e.g., YouTube).
2. Install a browser extension that can export cookies, such as "Get cookies.txt".
3. Export cookies from the target site and copy the exported text (Netscape format).

### Configure in LiveTV
1. Open the settings page and find "yt-dlp Cookies Content (optional)".
2. Paste the cookies.txt content and save.
3. Leave it empty and save to disable cookies.

### Security Notes
- Cookies are saved to `LIVETV_DATADIR/cookies.txt` with permission `600`.
- Only paste-in content is supported; custom paths or file uploads are not allowed to avoid path risks.

## New Features

### 1. Customizable M3U Playlist Filename
- Default: `lives.m3u`
- Can be customized in the configuration management section
- Changes take effect immediately without server restart

### 2. Customizable Channel Parameter
- Default: `c` (e.g., `live.m3u8?c=1`)
- Can be customized in the configuration management section

### 3. Custom Channel IDs
- Supports custom string IDs for channels (e.g., "news", "sports")
- Each channel can have a unique custom ID
- Displayed in the channel list

### 4. Log Output Mode
- Default: Standard output only
- Environment variable `LIVETV_LOG_FILE=1` enables file logging
- Logs stored in `./data/livetv.log`

### 5. Error Handling for Scheduled Caching
- Tracks failed channels
- Puts channels in cooldown after 3 consecutive failures
- Automatically retries after 24-hour cooldown

### 6. Automatic Directory Creation
- Creates data directory if it doesn't exist
- Ensures proper storage structure for logs and database

### 7. Version Number
- Displays version information on startup
- Current version: 1.0.0

### 8. Dynamic yt-dlp Parameters
- Changes to `ytdl_cmd` / `ytdl_args` / `ytdl_cookies` / `ytdl_timeout` take effect without restart
- Cache is cleared automatically so new parameters apply immediately

Document Translate by [DeepL](https://www.deepl.com/zh/translator)

nginx proxy set

```nginx
upstream  youtube {
        least_conn;
        server 127.0.0.1:9000 max_fails=3 fail_timeout=30s;
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
