package main

import (
	"fmt"

	"github.com/mdlayher/vsock"
)

func main() {

	cid, err := vsock.ContextID()
	var s string
	if err == nil {
		s = fmt.Sprint(cid)
	}

	fmt.Printf("CID is: %s\n", s)

}
