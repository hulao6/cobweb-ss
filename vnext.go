package main

import (
	"encoding/json"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	v2ray "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/router"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	v4 "github.com/v2fly/v2ray-core/v5/infra/conf/v4"
)

type VNext struct {
	core.App
}

func VNextRule(app core.App) *router.Rule {
	rule := &VNext{app}
	return &router.Rule{
		Condition:  rule,
		DynamicTag: rule,
	}
}

func (app *VNext) Apply(ctx routing.Context) bool {
	id := toID(ctx.GetUser())
	d, err := app.FindRecordById("devices", id)
	if err != nil {
		return false
	}
	vnext := d.GetString("vnext")
	if vnext == "" {
		return false
	}
	return true
}

func (app *VNext) PickOutbound(ctx routing.Context) (string, error) {
	id := toID(ctx.GetUser())
	d, err := app.FindRecordById("devices", id)
	if err != nil {
		return "", err
	}
	vnext := d.GetString("vnext")
	return vnext, nil
}

func toID(email string) string {
	id, _ := strings.CutSuffix(email, "@internal.invalid")
	return id
}

func getOutboundConfig(record *core.Record) (*v2ray.OutboundHandlerConfig, error) {
	var jsonConf v4.OutboundDetourConfig
	outbound := record.GetString("outbound")
	if err := json.Unmarshal([]byte(outbound), &jsonConf); err != nil {
		return nil, err
	}
	jsonConf.Tag = record.Id
	return jsonConf.Build()
}
