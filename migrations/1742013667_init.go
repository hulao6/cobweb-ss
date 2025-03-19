package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)
		users := try.To1(app.FindCollectionByNameOrId("users"))
		users.ListRule = types.Pointer(`id = @request.auth.id`)
		users.ViewRule = types.Pointer(`id = @request.auth.id`)
		users.CreateRule = types.Pointer(``)
		users.UpdateRule = nil
		users.DeleteRule = nil
		users.PasswordAuth.Enabled = false
		try.To(app.Save(users))

		devices := core.NewBaseCollection("devices", ID("devices"))
		devices.ListRule = types.Pointer(`@request.query.uuid = uuid`)
		devices.ViewRule = types.Pointer(`@request.query.uuid = uuid`)
		devices.CreateRule = nil
		devices.UpdateRule = nil
		devices.DeleteRule = nil
		devices.Fields.Add(
			&core.TextField{
				Name: "name", Id: ID("name"), System: true,
				Presentable: true,
			},
			&core.TextField{
				Name: "uuid", Id: ID("uuid"), System: true, // 在程序层面避免重复
				Required: true,
			},
			&core.NumberField{
				Name: "transmit_bytes", Id: ID("transmit_bytes"), System: true,
				OnlyInt: true,
			},
		)
		addUpdatedFields(&devices.Fields)
		devices.AddIndex("uuid", true, "uuid", "")
		try.To(app.Save(devices))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("rollback init todo")
	})
}
