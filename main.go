package main

import (
	"s3.abiao.me/server"
)

//func safeDirPath() {
//	current := time.Now()
//	year := strconv.Itoa(current.Year())
//	month := current.Month().String()
//	day := strconv.Itoa(current.Day())
//	fmt.Println(year, month, day)
//}

func main() {
	//safeDirPath()
	server.Start()
}
