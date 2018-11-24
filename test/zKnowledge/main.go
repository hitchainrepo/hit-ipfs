package main

import (
	"fmt"
	"math/rand"
)

func main(){
	//nodeHash := "QmdfYLM2jQRF6EMWNQwbMeTmqrxw1YAFA4ithj6KctVRZ8"
	//nodeInt, err := strconv.ParseInt(nodeHash, 10, 64)
	//if err != nil{
	//	panic(err)
	//}
	nodeId := 123
	rand.NewSource((int64(nodeId)))
	for i := 0; i < 10; i++{
		a := rand.Intn(100)
		fmt.Println(a)
	}
}
