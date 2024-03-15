package main

import (
  "Update-PTR-Record/module/config"
	"Update-PTR-Record/module/menu"
	"fmt"
)

func main() {
  cfg, err := FixDNSConfig.ReadConfig("config.yml")
  if err != nil {
      fmt.Println("Program exit when try to read config file, with error: ", err.Error())
      return
  }
  FixDNSMenu.Menu(cfg)
}