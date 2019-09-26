package main

import (
	"fmt"
	"github.com/spf13/viper"
	"runtime"
)

//var debug = true

func main() {
	getConfig()
	f_debug("In function")
}

func getConfig() {
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.SetDefault("debug", true)
	f_debug("In function")
	err := viper.ReadInConfig()
	if err != nil {
		bail(err)
	}
}

func bail(msg error) {
	f_debug("In function")
	panic(fmt.Errorf("Fatal error : %s \n", msg))
}

func f_debug(msg string) {
	if viper.Get("debug") == true {
		caller, _, _, _ := runtime.Caller(1)
		cname := runtime.FuncForPC(caller).Name()
		fmt.Println("DEBUG FROM: " + cname + " : " + msg)
	}
}
