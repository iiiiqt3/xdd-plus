package middleware

import (
	"fmt"
	"github.com/astaxie/beego/context"
)

var LogFilter = func(ctx *context.Context) {
	fmt.Printf("Request URI: %s\n", ctx.Input.URI())
}
