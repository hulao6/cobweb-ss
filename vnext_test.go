package main

import (
	"encoding/json"
	"testing"

	core "github.com/v2fly/v2ray-core/v5"
	v4 "github.com/v2fly/v2ray-core/v5/infra/conf/v4"
)

func TestVNext(t *testing.T) {
	var settings = []byte(`{
	"tag": "out-vmess",
	"protocol": "vmess",
	"streamSettings": {
		"network": "ws",
		"wsSettings": { "path": "/ray" }
	},
	"mux": { "enabled": true },
	"settings": {
		"vnext": [
			{
				"address": "127.0.0.1",
				"port": 10000,
				"users": [
					{
						"id": "b831381d-6324-4d53-ad4f-8cda48b30811",
						"alterId": 0
					}
				]
			}
		]
	}
}`)
	var c v4.OutboundDetourConfig
	json.Unmarshal(settings, &c)
	config, err := c.Build()
	if err != nil {
		t.Error(err)
		return
	}
	if false {
		core.AddOutboundHandler(nil, config)
	}
}
