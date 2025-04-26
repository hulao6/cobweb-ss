package main

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/shynome/cobweb/v3/migrations"
	"github.com/shynome/err0/try"
)

var Version = "dev"

var (
	wsPath   string
	wsListen string
	wsPort   string
)

func main() {
	app := pocketbase.New()
	app.RootCmd.Version = Version
	app.RootCmd.PersistentFlags().StringVar(&wsPath, "ws-path", "/ray", "v2ray websocket path")
	app.RootCmd.PersistentFlags().StringVar(&wsListen, "ws-listen", "./ws.socket", "v2ray listen unix socket filepath or addr")
	app.RootCmd.PersistentFlags().StringVar(&wsPort, "ws-port", "0", "v2ray websocket listen port")
	app.OnServe().BindFunc(initV2ray)
	app.OnBackupCreate().BindFunc(func(e *core.BackupEvent) error {
		e.Exclude = append(e.Exclude, "auxiliary.db", "auxiliary.db-shm", "auxiliary.db-wal")
		return e.Next()
	})
	try.To(app.Start())
}
