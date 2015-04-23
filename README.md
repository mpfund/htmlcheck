# htmlcheck
simple, fast and easy html checker in Go

``` Go
package main

import (
	"fmt"
	"github.com/BlackEspresso/htmlcheck"
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
	err := validater.ValidateHtmlString("<a href='http://google.com'>m</a>")
	if err != nil {
		fmt.Println(err.Error())
		return
	} else {
		fmt.Println("ok")
	}
	
	// second check
  // notice the missing / in the second <a>:
	err = validater.ValidateHtmlString("<a href='http://google.com'>m<a>")
	if err != nil {
		fmt.Println(err.Error())
		return
	} else {
		fmt.Println("ok")
	}
}
```

prints

```
ok
tag 'a' is not properly closed
```
