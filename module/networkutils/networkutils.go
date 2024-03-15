package FixDNS

import (
	"binary"
	"bufio"
	"errors"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

//convert Integer to string
func int2ip(nn uint32) string {
    ip := make(net.IP, 4)
    binary.BigEndian.PutUint32(ip, nn)
    
    return ip.String()
}

//if host didnt respond to echo request save him to file 
func AddDeadHostToLogFile(PathToLogFile string, ip string, ErrorMassage error) (error){
    fopen, err := os.OpenFile(PathToLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
    if err != nil{
        return err
    }   
    if _, err := fopen.WriteString("Echo response from " + ip + " exit with error:" + ErrorMassage.Error() + "\n"); err != nil {
        return err
    }
    
    return nil
}

//return array with 24 subnets
func GetSubnets(netCIDR string, subnetMaskSize int) ([]string, error){
    ip, ipNet, err := net.ParseCIDR(netCIDR)
    if err != nil {
        return nil, err
    }
    if !ip.Equal(ipNet.IP) {
        return nil, errors.New("netCIDR is not a valid network address")
    }
    netMaskSize, _ := ipNet.Mask.Size()
    if netMaskSize > int(subnetMaskSize) {
        return nil, errors.New("subnetMaskSize must be greater or equal than netMaskSize")
    }

    totalSubnetsInNetwork := math.Pow(2, float64(subnetMaskSize)-float64(netMaskSize))
    totalHostsInSubnet := math.Pow(2, 32-float64(subnetMaskSize))
    subnetIntAddresses := make([]uint32, int(totalSubnetsInNetwork))

    subnetIntAddresses[0] = binary.BigEndian.Uint32(ip.To4())
    for i := 1; i < int(totalSubnetsInNetwork); i++ {
        subnetIntAddresses[i] = subnetIntAddresses[i-1] + uint32(totalHostsInSubnet)
    }

   subnetCIDRs := make([]string, 0)
   for _, sia := range subnetIntAddresses {
        subnetCIDRs = append(
            subnetCIDRs,
            strings.TrimSuffix(int2ip(sia), ".0"),
         )
   }

   return subnetCIDRs, nil
}

//Return host state
func Ping (ipaddress string) (bool, error){
    args := "-c 1 -W 2 " + ipaddress
    cmd := exec.Command("ping", strings.Split(args, " ")...)
  
    output, _ := cmd.StdoutPipe()
    cmd.Start()

    scanner := bufio.NewScanner(output)

    for scanner.Scan() {
        m := scanner.Text()
        if flag, _ := regexp.MatchString("100% packet loss", m); flag {  
            return false, errors.New("100% packet loss\n")
        }

    }
    return true, nil
}
