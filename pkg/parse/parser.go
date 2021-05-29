package parse

import (
	"errors"
	"fmt"
	"github.com/watertreestar/m3u8-dl/pkg/tool"
	"io/ioutil"
	"net/url"
)

type Result struct {
	URL  *url.URL
	M3u8 *M3u8
	Keys map[int]string
}

// ParseFromNet Gets parse result from a url link
func ParseFromNet(urlLink string) (*Result, error) {
	u, err := url.Parse(urlLink)
	if err != nil {
		return nil, err
	}

	link := u.String()
	body, err := tool.Get(link)
	if err != nil {
		return nil, fmt.Errorf("request m3u8 URL failed: %s", err.Error())
	}

	//noinspection GoUnhandledErrorResult
	defer body.Close()
	m3u8, err := parse(body)
	if err != nil {
		return nil, err
	}
	// 如果是master list,选取第一个作为play list的m3u8地址
	if len(m3u8.MasterPlaylist) != 0 {
		sf := m3u8.MasterPlaylist[0]
		return ParseFromNet(tool.ResolveURL(u, sf.URI))
	}
	if len(m3u8.Segments) == 0 {
		return nil, errors.New("can not found any TS file description")
	}

	if len(m3u8.Segments) == 0 {
		return nil, errors.New("can not found any TS file description")
	}

	result := &Result{
		URL:  u,
		M3u8: m3u8,
		Keys: make(map[int]string),
	}

	// resolve keys for encrypt decode
	// 如果ts片段是加密的,获取每个片段的密钥
	for idx, key := range m3u8.Keys {
		switch {
		case key.Method == "" || key.Method == CryptMethodNONE:
			continue
		case key.Method == CryptMethodAES:
			// Request URL to extract decryption key
			keyURL := key.URI
			keyURL = tool.ResolveURL(u, keyURL)
			resp, err := tool.Get(keyURL)
			if err != nil {
				return nil, fmt.Errorf("extract key failed: %s", err.Error())
			}
			keyByte, err := ioutil.ReadAll(resp)
			_ = resp.Close()
			if err != nil {
				return nil, err
			}
			fmt.Println("decryption key: ", string(keyByte))
			result.Keys[idx] = string(keyByte)
		default:
			return nil, fmt.Errorf("unknown or unsupported cryption method: %s", key.Method)
		}
	}
	return result, nil
}
