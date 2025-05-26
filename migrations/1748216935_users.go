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

		// 隐藏 users 表
		users := try.To1(app.FindCollectionByNameOrId("users"))
		users.System = true
		// 强制隐藏 users 表
		try.To(app.SaveNoValidate(users))

		return nil
	}, func(app core.App) error {
		return fmt.Errorf("users collections hide rollback todo")
	})
}
