package jobs

import (
	"app/lib"
	"fmt"
	"os"
)

var _ = lib.RegisterJob("secrets", func(c *lib.Ctx, args lib.J) {
	for _, v := range os.Environ() {
		fmt.Println(v)
	}
})

var _ = lib.RegisterJob("secrets-encrypt", func(c *lib.Ctx, args lib.J) {
	fmt.Println("$e1$" + lib.SecretsEncrypt(os.Getenv("SECRET"), os.Args[2]))
})
