# Windows Shutdown Hook

Windows shutdown hook implement in pure Go.

## Install

```shell
go get -u github.com/chengxuncc/shutdownhook
```

## Example

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/chengxuncc/shutdownhook"
)

func main() {
	f, err := os.Create("shutdown.log")
	if err != nil {
		panic(err)
	}
	err = shutdownhook.New(func() {
		fmt.Fprintf(f, "%s shutdown\n", time.Now())
		f.Close()
	})
	if err != nil {
		panic(err)
	}
	select {}
}
```