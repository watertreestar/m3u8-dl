# m3u8

使用golang实现m3u8视频文件的下载

## 前言

什么是m3u8，接触过直播的人应该知道，HLS作为一种拉流的传输协议，我们时常也可以在浏览器上看到直播，这种使用了HLS，也就通过HTTP来传输HLS文件。

在一些小电影，电视剧的网站上，我们也可以发现他的踪迹。浏览器F12打开开发者工具，网络面板中看到一直有连续的ts文件请求，就是它了

M3U8 —— Unicode 版本的 M3U（Moving Picture Experts Group Audio Layer 3 Uniform Resource Locator），
使用了 UTF-8 编码，是 HLS（HTTP Living Stream，苹果公司基于 HTTP 实现的媒体流传输协议）协议的一部分，作为媒体文件描述清单，另外一部分为 TS（Transport Stream，传输流） 媒体文件。

M3U8 文件使用特定标签描述了媒体流的详细信息，包括时长、版本、编码、音频、字幕、播放列表、加密等。M3U8 媒体播放列表中保存了 TS 媒体文件的路径列表

例如，一个m3u8文件：
```
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:8
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:5.004,
/20190319/DnYZi3eA/800kb/hls/imaOxa8299000.ts
#EXTINF:4.17,
/20190319/DnYZi3eA/800kb/hls/imaOxa8299001.ts
#EXTINF:6.005,
```

## 前置知识

那么，在开始编码前，我们要知道一个m3u8文件的构成:

### 1. m3u8类型

当 M3U8 文件作为媒体播放列表（Media Playlist）时，其内部信息记录的是一系列媒体片段资源，顺序播放该片段资源，即可完整展示多媒体资源。其格式如下所示
```
#EXTM3U
#EXT-X-TARGETDURATION:10

#EXTINF:9.009,
http://media.example.com/first.ts
#EXTINF:9.009,
http://media.example.com/second.ts
#EXTINF:3.003,
http://media.example.com/third.ts
#EXT-X-ENDLIST
```

> 有些 TS 文件是经过加密处理的，下载下来无法直接播放，需要对 TS 数据进行解密，METHOD 为加密方式，一般为 AES-128 或者 NONE。如果为 AES-128 则有 URI 给定秘钥的存放位置，
> 部分加密还是用了 IV 偏移向量，因此在解密的时候需要格外注意，记得一起使用 IV 来进行解密。如果 METHOD 为 NONE 则表示没有加密，默认可以不声明 #EXT-X-KEY，NONE 的情况下不能出现 URI 和 IV


当 M3U8 作为主播放列表（Master Playlist）时，其内部提供的是同一份媒体资源的多份流列表资源。其格式如下所示：
```
#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=150000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/low/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=240000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/lo_mid/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=440000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/hi_mid/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=640000,RESOLUTION=640x360,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/high/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=64000,CODECS="mp4a.40.5"
http://example.com/audio/index.m3u8
#EXT-X-ENDLIST
```



### 2. m3u8的基本属性

- #EXTM3U                    M3U8文件头，必须放在第一行;
- #EXT-X-MEDIA-SEQUENCE      第一个TS分片的序列号，一般情况下是0，但是在直播场景下，这个序列号标识直播段的起始位置; #EXT-X-MEDIA-SEQUENCE:0
- #EXT-X-TARGETDURATION      每个分片TS的最大的时长;   #EXT-X-TARGETDURATION:10     每个分片的最大时长是 10s
- #EXT-X-ALLOW-CACHE         是否允许cache;          #EXT-X-ALLOW-CACHE:YES      #EXT-X-ALLOW-CACHE:NO    默认情况下是YES
- #EXT-X-ENDLIST             M3U8文件结束符；
- #EXTINF                    extra info，分片TS的信息，如时长，带宽等；一般情况下是    #EXTINF:<duration>,[<title>] 后面可以跟着其他的信息，逗号之前是当前分片的ts时长，分片时长 移动要小于 #EXT-X-TARGETDURATION 定义的值；
- #EXT-X-VERSION             M3U8版本号
- #EXT-X-DISCONTINUITY       该标签表明其前一个切片与下一个切片之间存在中断。下面会详解
- #EXT-X-PLAYLIST-TYPE       表明流媒体类型；
- #EXT-X-KEY                 是否加密解析，    #EXT-X-KEY:METHOD=AES-128,URI="https://priv.example.com/key.php?r=52"    加密方式是AES-128,秘钥需要请求   https://priv.example.com/key.php?r=52  ，请求回来存储在本地；


### 3. 怎么判断是否是m3u8直播


1.判断是否存在 #EXT-X-ENDLIST
对于一个M3U8文件，如果结尾不存在 #EXT-X-ENDLIST，那么一定是 直播，不是点播；

2.判断 #EXT-X-PLAYLIST-TYPE 类型
'#EXT-X-PLAYLIST-TYPE' 有两种类型，

- VOD 即 Video on Demand，表示该视频流为点播源，因此服务器不能更改该 M3U8 文件；

- EVENT 表示该视频流为直播源，因此服务器不能更改或删除该文件任意部分内容（但是可以在文件末尾添加新内容）
（注：VOD 文件通常带有 EXT-X-ENDLIST 标签，因为其为点播片源，不会改变；而 EVEVT 文件初始化时一般不会有 EXT-X-ENDLIST 标签，暗示有新的文件会添加到播放列表末尾，因此也需要客户端定时获取该 M3U8 文件，以获取新的媒体片段资源，直到访问到 EXT-X-ENDLIST 标签才停止）
  

### 4. 多码率

例如一个master list中：
```
#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=150000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/low/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=240000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/lo_mid/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=440000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/hi_mid/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=640000,RESOLUTION=640x360,CODECS="avc1.42e00a,mp4a.40.2"
http://example.com/high/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=64000,CODECS="mp4a.40.5"
http://example.com/audio/index.m3u8
#EXT-X-ENDLIST
```

通过不同码率选择上面不同的m3u8文件，然后在获取ts片段



## 实现

根据上面的内容，我们在心里可以大概有一个m3u8文件的结构

![image-20210529132616827](https://cdn.jsdelivr.net/gh/watertreestar/CDN@master/picimage-20210529132616827.png)

接下来的代码实现中也会创建对应的数据结构,我们需要从string或者io流中解析出上述的结构，以便进行后续的操作







