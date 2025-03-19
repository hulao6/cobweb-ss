package main

import (
	"context"
	_ "embed"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/shynome/err0/try"
	v2ray "github.com/v2fly/v2ray-core/v5"
	v2inbound "github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/proxy"
)

var sm stats.Manager

func TestMain(m *testing.M) {
	config := try.To1(v2ray.LoadConfig("json", strings.NewReader(v2rayConf)))
	srv := try.To1(v2ray.New(config))
	try.To(srv.Start())
	defer srv.Close()
	sm = srv.GetFeature(stats.ManagerType()).(stats.Manager)
	var ibm v2inbound.Manager = srv.GetFeature(v2inbound.ManagerType()).(v2inbound.Manager)
	ctx := context.Background()
	p := try.To1(ibm.GetHandler(ctx, "main"))
	um, ok := p.(proxy.GetInbound).GetInbound().(proxy.UserManager)
	log.Println(um, ok)

	c := sm.GetCounter("user>>>test@test.invalid>>>traffic>>>uplink")
	log.Println(sm, c)
	m.Run()
}

func TestV2ray(t *testing.T) {

	config := try.To1(v2ray.LoadConfig("json", strings.NewReader(testV2rayClientConf)))
	v2 := try.To1(v2ray.New(config))
	defer v2.Close()
	try.To(v2.Start())

	proxy := http.ProxyURL(try.To1(url.Parse("socks5://127.0.0.1:1080")))
	client := &http.Client{
		Transport: &http.Transport{Proxy: proxy},
	}
	req := try.To1(http.NewRequest(http.MethodGet, "http://myip.ipip.net", nil))
	resp := try.To1(client.Do(req))
	defer resp.Body.Close()
	body := string(try.To1(io.ReadAll(resp.Body)))
	t.Log(body)

	c := sm.GetCounter("user>>>test@test.invalid>>>traffic>>>uplink")
	log.Println(sm, c)
}

//go:embed client.jsonc
var testV2rayClientConf string
