package httpproxy

import "fmt"

var Name string = "httpproxy"
var Version string = "unkown"

func ShowVersion() {
	fmt.Printf("%s/%s\n", Name, Version)
}
