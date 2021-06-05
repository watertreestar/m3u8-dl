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

- `#EXTM3U`                    M3U8文件头，必须放在第一行;
- `#EXT-X-MEDIA-SEQUENCE`      第一个TS分片的序列号，一般情况下是0，但是在直播场景下，这个序列号标识直播段的起始位置; #EXT-X-MEDIA-SEQUENCE:0
- `#EXT-X-TARGETDURATION`      每个分片TS的最大的时长;   #EXT-X-TARGETDURATION:10     每个分片的最大时长是 10s
- `#EXT-X-ALLOW-CACHE`         是否允许cache;          #EXT-X-ALLOW-CACHE:YES      #EXT-X-ALLOW-CACHE:NO    默认情况下是YES
- `#EXT-X-ENDLIST `            M3U8文件结束符；
- `#EXTINF`                    extra info，分片TS的信息，如时长，带宽等；一般情况下是    #EXTINF:<duration>,[<title>] 后面可以跟着其他的信息，逗号之前是当前分片的ts时长，分片时长 移动要小于 #EXT-X-TARGETDURATION 定义的值；
- `#EXT-X-VERSION`             M3U8版本号
- `#EXT-X-DISCONTINUITY`       该标签表明其前一个切片与下一个切片之间存在中断。下面会详解
- `#EXT-X-PLAYLIST-TYPE`       表明流媒体类型；
- `#EXT-X-KEY `                是否加密解析，    #EXT-X-KEY:METHOD=AES-128,URI="https://priv.example.com/key.php?r=52"    加密方式是AES-128,秘钥需要请求   https://priv.example.com/key.php?r=52  ，请求回来存储在本地；


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



## 代码

### 1. 数据结构定义

上一篇分析了一个m3u8文件的结构，那我们现在就从这个文件中读取出计算机能够识别的结构。

这个文件的parse非常简单，不涉及到复杂的语法，现在我们需要覆盖的范围也就只是上一篇提到的几个，所以关键字也没有几个。那么我们就可以按行读取出来，
每一行就是一个字符串，根据这个字符串的头部来判断是一个什么结构，读取的过程中顺便判断这个文件是不是合法的，很容易

首先我们需要一个struct来表示一个m3u8文件
```go
type M3u8 struct {
	Version        int8              // EXT-X-VERSION:version
	MediaSequence  uint64            // Default 0, #EXT-X-MEDIA-SEQUENCE:sequence
	Segments       []*Segment        // Define a Play List
	MasterPlaylist []*MasterPlaylist // Define a Master Play List
	Keys           map[int]*Key      // Keys for per segment
	EndList        bool              // #EXT-X-ENDLIST
	PlaylistType   PlaylistType      // VOD or EVENT
	TargetDuration float64           // #EXT-X-TARGETDURATION:duration
}
```

上面看到有Segment和MasterPlayList,这两个二选一，也就是上篇说的一个m3u8可以是一个MasterPlayList来提供多码率，从MasterPalyList中可以选择一个
特定的码率，然后拿到一个新的m3u8文件，这个m3u8中包含了多个Segment，Segment包含了ts片段（也就是一个代表ts的URI）.还看到一个Key数组，这个是对于每一个
Segment的加密密钥

```go
// Segment
// #EXTINF:10.000000,
// 5dd92bfb879c6421d7281c769f0f8c93-4.ts
type Segment struct {
	URI      string
	KeyIndex int
	Title    string  // #EXTINF: duration,<title>
	Duration float32 // #EXTINF: duration,<title>
	Length   uint64  // #EXT-X-BYTERANGE: length[@offset]
	Offset   uint64  // #EXT-X-BYTERANGE: length[@offset]
}

// MasterPlaylist
// #EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=240000,RESOLUTION=416x234,CODECS="avc1.42e00a,mp4a.40.2"
type MasterPlaylist struct {
	URI        string
	BandWidth  uint32
	Resolution string
	Codecs     string
	ProgramID  uint32
}

// Key
// #EXT-X-KEY:METHOD=AES-128,URI="key.key"
type Key struct {
	// 'AES-128' or 'NONE'
	// If the encryption method is NONE, the URI and the IV attributes MUST NOT be present
	Method CryptMethod
	URI    string
	IV     string
}
```

### 2. parser

定义好这个结构后，就需要从文件中解析出这个结构，我们要做的就是从一个文件中读取一行，把这一行作为字符串，判断字符串的头部是什么开始的，如果匹配上了，就进一步处理
比如我们遇到一行
```
#EXTINF:10.000000,hello
```
我们就可以认为这是一个segment的的duration和title定义,那么我们就可以按照以下的方式来解析
```go
case strings.HasPrefix(line, "#EXTINF:"):
    if extInf {
        return nil, fmt.Errorf("duplicate EXTINF: %s, line: %d", line, i+1)
    }
    if seg == nil {
        seg = new(Segment)
    }
    var s string
    if _, err := fmt.Sscanf(line, "#EXTINF:%s", &s); err != nil {
        return nil, err
    }
    if strings.Contains(s, ",") {
        split := strings.Split(s, ",")
        seg.Title = split[1]
        s = split[0]
    }
    df, err := strconv.ParseFloat(s, 32)
    if err != nil {
        return nil, err
    }
    seg.Duration = float32(df)
    seg.KeyIndex = keyIndex
    extInf = true
```

整个parse过程就类似于上面这种，通过for + switch-case 来实现。其实就是一个简单的词法分析器，由于这里我们没有做语法分析，所以对于错误我们不能发现。
更加完善的词法分析可以了解编译原理的知识，或者通过有限自动机来实现

### 3. download

完成分析，构造出我们要的结构以后，就可以来进行下载了，通过http请求，然后保存每一个ts片段，最后我们把所有的ts片段合并成一个文件便完成了下载。
为了加快下载的速度，当然不能少了协程

关键下载的代码：
```go
var wg sync.WaitGroup
for {
    tsIdx, end, err := d.next()
    if err != nil {
        if end {
            break
        }
        continue
    }
    wg.Add(1)
    go func(idx int) {
        defer wg.Done()
        if err := d.download(idx); err != nil {
            // Back into the queue, retry request
            fmt.Printf("[failed] %s\n", err.Error())
            if err := d.back(idx); err != nil {
                fmt.Printf(err.Error())
            }
        }
    }(tsIdx)
}
wg.Wait()
if err := d.merge(); err != nil {
    return err
}
return nil
```

这个工具整体来说比较简单，这里这是提供一种思路，本身就很容易通过任何一门语言来实现

在做这个过程中，我也参考了别人的思路和代码。








