package main

import (
	"github.com/pocketbase/pocketbase"
	_ "github.com/shynome/cobweb/v3/migrations"
	"github.com/shynome/err0/try"
)

var wspath string
var wsfilepath string
var Version = "dev"

func main() {
	app := pocketbase.New()
	app.RootCmd.Version = Version
	app.RootCmd.PersistentFlags().StringVar(&wspath, "wspath", "/ray", "v2ray websocket path")
	app.RootCmd.PersistentFlags().StringVar(&wsfilepath, "wsfilepath", "./ws.socket", "v2ray unix socket filepath")
	app.OnServe().BindFunc(initV2ray)
	try.To(app.Start())
}
