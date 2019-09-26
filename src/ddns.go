package main

import (
	"fmt"
	"runtime"
	"github.com/spf13/viper"
)
var debug = true

func main()  {
	f_debug("In function")
	getConfig()
}
func funcName() string {
	caller, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(caller).Name()
}

func getConfig() {
	f_debug("In function")
	viper.SetConfigName("ddns")
	viper.AddConfigPath("~/.ddns")
	err := viper.ReadInConfig()
	if err != nil { bail("err")}
}

func bail(msg string) {
	f_debug("In function")
	panic(fmt.Errorf("Fatal error : %s \n", msg))
}

func f_debug(msg string) {
	if debug == true {
		caller, _, _, _ := runtime.Caller(1)
		cname := runtime.FuncForPC(caller).Name()
		fmt.Println("DEBUG FROM: " + cname + " : " + msg )
	}
}