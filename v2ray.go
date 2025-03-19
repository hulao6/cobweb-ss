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
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	v2ray "github.com/v2fly/v2ray-core/v5"
	v2stats "github.com/v2fly/v2ray-core/v5/app/stats"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/proxy/vmess"
)

//go:embed server.jsonc
var v2rayConf string

var (
	devicesTable = "devices"
)

func initV2ray(e *core.ServeEvent) (err error) {
	defer err0.Then(&err, nil, nil)
	socket := try.To1(filepath.Abs(wsfilepath))
	if _, err := os.Stat(socket); err == nil {
		try.To(os.Remove(socket))
		os.Remove(socket + ".lock")
	}

	replacer := strings.NewReplacer(
		"debug", "error",
		"10000", "0",
		"127.0.0.1", socket,
		"/ray", wspath,
	)
	v2conf := replacer.Replace(v2rayConf)
	cfg := try.To1(v2ray.LoadConfig("json", strings.NewReader(v2conf)))
	v2 := try.To1(v2ray.New(cfg))

	ibm := v2.GetFeature(inbound.ManagerType()).(inbound.Manager)

	ctx := context.Background()
	h := try.To1(ibm.GetHandler(ctx, "main"))
	um := &userManager{h.(proxy.GetInbound).GetInbound().(proxy.UserManager)}
	if dev := e.App.IsDev(); !dev {
		um.RemoveUser(ctx, "test@test.invalid")
	}
	v2stats := v2.GetFeature(stats.ManagerType()).(*v2stats.Manager)

	devices := try.To1(e.App.FindAllRecords(devicesTable))
	{
		logger := e.App.Logger()
		for _, d := range devices {
			if err := um.addRecord(ctx, d); err != nil {
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
		try.To(um.addRecord(ctx, e.Record))
		return nil
	})
	e.App.OnRecordDeleteRequest(devicesTable).BindFunc(func(e *core.RecordRequestEvent) (err error) {
		defer err0.Then(&err, nil, nil)
		ctx := e.Request.Context()
		try.To(um.removeRecord(ctx, e.Record))
		return e.Next()
	})

	try.To(v2.Start())
	e.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		return v2.Close()
	})

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.Dial("unix", socket)
			return conn, err
		},
	}
	u := try.To1(url.Parse("http://v2ray.com"))
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = transport
	e.Router.Any(wspath, func(e *core.RequestEvent) (err error) {
		proxy.ServeHTTP(e.Response, e.Request)
		return nil
	})
	return e.Next()
}

type userManager struct {
	proxy.UserManager
}

func (um userManager) addRecord(ctx context.Context, device *core.Record) error {
	idStr := device.GetString("uuid")
	id, err := uuid.ParseString(idStr)
	if err != nil {
		return err
	}
	acc := &vmess.MemoryAccount{
		ID:       protocol.NewID(id),
		Security: protocol.SecurityType_AUTO,
	}
	user := &protocol.MemoryUser{
		Account: acc,
		Email:   vmessEmail(device.Id),
		Level:   0,
	}
	return um.AddUser(ctx, user)
}

func (um userManager) removeRecord(ctx context.Context, device *core.Record) error {
	email := vmessEmail(device.Id)
	return um.RemoveUser(ctx, email)
}

func vmessEmail(id string) string {
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
