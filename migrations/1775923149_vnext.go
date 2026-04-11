package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
)

func init() {
	migrations.Register(func(app core.App) (err error) {
		defer err0.Then(&err, nil, nil)

		vnext := core.NewBaseCollection("vnext", ID("vnext"))
		vnext.Fields.Add(
			&core.TextField{
				Name: "name", Id: ID("name"), System: true,
				Presentable: true,
			},
			&core.JSONField{
				Name: "outbound", Id: ID("outbound"), System: true,
				Required: true,
			},
		)
		addUpdatedFields(&vnext.Fields)
		try.To(app.Save(vnext))

		devices := try.To1(app.FindCollectionByNameOrId("devices"))
		devices.Fields.AddAt(getFieldIndex(devices, "created"),
			&core.RelationField{
				Name: "vnext", Id: ID("vnext"), System: true,
				CollectionId: vnext.Id, MaxSelect: 1,
			},
		)
		try.To(app.Save(devices))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("rollback init todo")
	})
}
