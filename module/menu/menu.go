package FixDNSMenu

import (
	"fmt"
	"Update-PTR-Record/module/fixdns"
	"Update-PTR-Record/module/config"
)

func Menu (cfg *FixDNSConfig.Config) {

fmt.Print("Enter CIDR for PTR update -> ")
  var CIDR string
  _,_ = fmt.Scanln(&CIDR)
  fmt.Println()

  sshconfig := FixDNS.SSHConfig{
    SSHUserName:                   cfg.SSH.User,
    SSHKeyPass:                    cfg.SSH.PSKey,
    PathToSSHKey:                  cfg.SSH.PathToKey,
    PathToAccsessDeniedLogFile:    cfg.SSH.PathToAccsessDeniedLogFile,
  }

  pdnsconfig := FixDNS.PDNSConfig{
    HostName:                     cfg.PDNS.Host,
    ApiKey:                       cfg.PDNS.ApiKey,
    NameServers:                  cfg.PDNS.NameServers,
    PathToDeadHostsLogFile:       cfg.PDNS.PathToDeadHostsLogFile,
  }

  pdns := FixDNS.InitPowerDNS(&pdnsconfig)

//simple menu for start
  fmt.Print("------------------------- HELP -------------------------\nP - print records\nE - update PTR record\n------------------------- HELP -------------------------\n")
  fmt.Print("Enter mode: ")
  var mode string
  _,_ = fmt.Scanln(&mode)
  fmt.Println()

  switch mode {
  case "P":
    fmt.Print("------------------------- PRINT ------------------------\n\n")

    fmt.Print("Record type (default: A): ")
    var option string
    _,_ = fmt.Scanln(&option)
    switch mode {
    default:
      pdns.Records.PrintA(CIDR)
      return
    }

  case "E":
    fmt.Print("------------------------- EDIT -------------------------\n\n")
    pdns.Tools.MakeMeFeelGood(CIDR, &sshconfig)
  default:
    fmt.Println("HELP MENU")
    return
  }
}