package main

import (
	"github.com/pocketbase/pocketbase"
	_ "github.com/shynome/cobweb/v3/migrations"
	"github.com/shynome/err0/try"
)

var wspath string

func main() {
	app := pocketbase.New()
	app.RootCmd.PersistentFlags().StringVar(&wspath, "wspath", "/ray", "v2ray websocket path")
	app.OnServe().BindFunc(initV2ray)
	try.To(app.Start())
}
