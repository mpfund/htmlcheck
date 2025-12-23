# htmlcheck
simple, fast and easy html checker in Go

``` Go
package main

import (
	"fmt"
	"github.com/mpfund/htmlcheck"
)

func main() {
	validater := htmlcheck.Validator{}

	validLink := htmlcheck.ValidTag{
		Name:          "a",
		Attrs:         []string{"href", "target", "id"},
		IsSelfClosing: false,
	}

	validater.AddValidTag(validLink)
	// first check
	errors := validater.ValidateHtmlString("<a href='http://google.com'>m</a>")
	if len(errors) == 0 {
		fmt.Println("ok")
	} else {
		fmt.Println(errors)
	}

	// second check
	// notice the missing / in the second <a>:
	errors = validater.ValidateHtmlString("<a href='http://google.com'>m<a>")
	if len(errors) == 0 {
		fmt.Println("ok")
	} else {
		fmt.Println(errors)
	}
}
```

prints

```
ok
tag 'a' is not properly closed
```
