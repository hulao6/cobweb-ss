package main

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	v2ray "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/router"
	v2stats "github.com/v2fly/v2ray-core/v5/app/stats"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/proxy/trojan"
	"github.com/v2fly/v2ray-core/v5/proxy/vmess"
)

//go:embed server.jsonc
var v2rayConf string

var (
	devicesTable = "devices"
)

func initV2ray(e *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)

	if wsPort == "0" {
		wsListen = try.To1(filepath.Abs(wsListen))
		if _, err := os.Stat(wsListen); err == nil {
			try.To(os.Remove(wsListen))
			os.Remove(wsListen + ".lock")
		}
	}

	rr := []string{
		"10000", wsPort,
		"127.0.0.1", wsListen,
		"/ray", wsPath,
	}
	if dev := e.App.IsDev(); !dev {
		rr = append(rr,
			"debug", "error",
		)
	}
	if trojanPort != "" {
		rr = append(rr,
			"10001", trojanPort,
			"/trojan-ray", trojanPath,
		)
	}
	replacer := strings.NewReplacer(rr...)
	v2conf := replacer.Replace(v2rayConf)
	cfg := try.To1(v2ray.LoadConfig("json", strings.NewReader(v2conf)))
	if trojanPort == "" {
		cfg.Inbound = slices.DeleteFunc(cfg.Inbound, func(inbound *v2ray.InboundHandlerConfig) bool {
			return inbound.Tag == "trojan"
		})
	}
	v2 := try.To1(v2ray.New(cfg))

	ibm := v2.GetFeature(inbound.ManagerType()).(inbound.Manager)

	ctx := context.Background()
	mum := &multiUserManager{}
	h := try.To1(ibm.GetHandler(ctx, "main"))
	mum.um[0] = h.(proxy.GetInbound).GetInbound().(proxy.UserManager)
	if trojanPort != "" {
		h := try.To1(ibm.GetHandler(ctx, "trojan"))
		mum.um[1] = h.(proxy.GetInbound).GetInbound().(proxy.UserManager)
	}
	if dev := e.App.IsDev(); !dev {
		for _, um := range mum.um {
			if um == nil {
				continue
			}
			um.RemoveUser(ctx, "test@test.invalid")
		}
	}
	v2stats := v2.GetFeature(stats.ManagerType()).(*v2stats.Manager)

	rm := v2.GetFeature(routing.RouterType()).(*router.Router)
	rm.AddRule(VNextRule(e.App))
	vnextList := try.To1(e.App.FindAllRecords("vnext"))
	for _, record := range vnextList {
		c := try.To1(getOutboundConfig(record))
		try.To(v2ray.AddOutboundHandler(v2, c))
	}
	addOutbound := func(e *core.RecordEvent) error {
		if _, err := getOutboundConfig(e.Record); err != nil {
			return err
		}
		if err := e.Next(); err != nil {
			return err
		}
		c, err := getOutboundConfig(e.Record)
		if err != nil {
			return err
		}
		return v2ray.AddOutboundHandler(v2, c)
	}
	e.App.OnRecordCreate("vnext").BindFunc(addOutbound)
	e.App.OnRecordUpdate("vnext").BindFunc(addOutbound)
	e.App.OnRecordDelete("vnext").BindFunc(func(e *core.RecordEvent) error {
		if err := v2ray.RemoveOutboundHandler(v2, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	})

	devices := try.To1(e.App.FindAllRecords(devicesTable))
	{
		logger := e.App.Logger()
		for _, d := range devices {
			if err := mum.addRecord(ctx, d); err != nil {
				logger.Error("add device failed", "device", d, "error", err)
			}
		}
	}
	// 每小时收集一次
	e.App.Cron().MustAdd("collect_usages", "0 * * * *", func() {
		collectUsages(e.App, v2stats)
	})

	e.App.OnRecordCreateRequest(devicesTable).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		idStr := e.Record.GetString("uuid")
		if _, err := uuid.ParseString(idStr); err != nil {
			return apis.NewBadRequestError("uuid 解析失败", err)
		}
		try.To(e.Next())
		ctx := e.Request.Context()
		try.To(mum.addRecord(ctx, e.Record))
		return nil
	})
	e.App.OnRecordDeleteRequest(devicesTable).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		ctx := e.Request.Context()
		try.To(mum.removeRecord(ctx, e.Record))
		return e.Next()
	})

	try.To(v2.Start())
	e.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		return v2.Close()
	})

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.Dial("unix", wsListen)
			return conn, err
		},
	}
	if wsPort != "0" {
		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := net.Dial("tcp", "127.0.0.1:"+wsPort)
				return conn, err
			},
		}
	}
	u := try.To1(url.Parse("http://v2ray.com"))
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = transport
	e.Router.Any(wsPath, func(e *core.RequestEvent) (err error) {
		proxy.ServeHTTP(e.Response, e.Request)
		return nil
	})
	return e.Next()
}

type multiUserManager struct {
	um [2]proxy.UserManager // [0]vmess [1]trojan
}

func (mum multiUserManager) addRecord(ctx context.Context, device *core.Record) error {
	idStr := device.GetString("uuid")
	id, err := uuid.ParseString(idStr)
	if err != nil {
		return err
	}
	if um := mum.um[0]; um != nil {
		acc := &vmess.MemoryAccount{
			ID:       protocol.NewID(id),
			Security: protocol.SecurityType_AUTO,
		}
		user := &protocol.MemoryUser{
			Account: acc,
			Email:   protocolEmail(device.Id),
			Level:   0,
		}
		if err := um.AddUser(ctx, user); err != nil {
			return err
		}
	}
	if um := mum.um[1]; um != nil {
		acc, _ := (&trojan.Account{Password: idStr}).AsAccount()
		user := &protocol.MemoryUser{
			Account: acc,
			Email:   protocolEmail(device.Id),
			Level:   0,
		}
		if err := um.AddUser(ctx, user); err != nil {
			return err
		}
	}
	return nil
}

func (mum multiUserManager) removeRecord(ctx context.Context, device *core.Record) error {
	email := protocolEmail(device.Id)
	for _, um := range mum.um {
		if um == nil {
			continue
		}
		if err := um.RemoveUser(ctx, email); err != nil {
			return err
		}
	}
	return nil
}

func protocolEmail(id string) string {
	return fmt.Sprintf("%s@internal.invalid", id)
}

func collectUsages(app core.App, v2stats *v2stats.Manager) (err error) {
	defer err0.Then(&err, nil, nil)
	cs := map[string]*Counter{}
	// user>>>test@test.invalid>>>traffic>>>uplink
	// user>>>test@test.invalid>>>traffic>>>downlink
	v2stats.VisitCounters(func(s string, c stats.Counter) (cont bool) {
		cont = true
		l := strings.Split(s, ">>>")
		if len(l) != 4 {
			return
		}
		if l[0] != "user" {
			return
		}
		if !strings.HasSuffix(l[1], "@internal.invalid") {
			return
		}
		did := strings.TrimSuffix(l[1], "@internal.invalid")
		counter, ok := cs[did]
		if !ok {
			counter = &Counter{}
			cs[did] = counter
		}
		switch l[3] {
		case "uplink":
			counter.uplink = c.Set(0)
		case "downlink":
			counter.downlink = c.Set(0)
		}
		return
	})
	logger := app.Logger()
	for did, c := range cs {
		if c.downlink == 0 && c.uplink == 0 {
			continue
		}
		logger := logger.With("id", did)
		if err := app.RunInTransaction(func(tx core.App) (err error) {
			defer err0.Then(&err, nil, nil)
			device := try.To1(tx.FindRecordById(devicesTable, did))
			transmit := int64(device.GetInt("transmit_bytes"))
			transmit += c.uplink + c.downlink
			device.Set("transmit_bytes", transmit)
			try.To(tx.Save(device))
			return nil
		}); err != nil {
			logger.Error("update usage failed", "error", err)
		}
	}
	return nil
}

type Counter struct {
	uplink   int64
	downlink int64
}
