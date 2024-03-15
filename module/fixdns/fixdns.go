package FixDNS

import (
  "fmt"
  "log"
  "strconv"
  PowerDNS     "Update-PTR-Record/module/powerdns"
  NetworkUtils "Update-PTR-Record/module/networkutils"
  SSHBot       "Update-PTR-Record/module/sshbot"
)

type service struct {
    client *PDNSClient
}

type RecordsService service
type ToolsService service

type PDNSConfig struct {
    HostName                     string
    ApiKey                       string
    NameServers                  []string
    PathToDeadHostsLogFile       string
}

type PDNSClient struct {
    PDNSConfig *PDNSConfig
    common service
    Powerdns *PowerDNS.Powerdns
    Records *RecordsService
    Tools *ToolsService
}

type SSHConfig struct {
    SSHUserName                 string
    Address                     string
    SSHKeyPass                  string
    PathToSSHKey                string
    PathToAccsessDeniedLogFile  string
}

func InitPowerDNS (pdnsconfig *PDNSConfig) *PDNSClient {
    client := &PDNSClient{
        PDNSConfig: pdnsconfig,
        Powerdns: PowerDNS.NewPowerdns(pdnsconfig.HostName, pdnsconfig.ApiKey, pdnsconfig.NameServers),
    }

    client.common.client = client
    client.Records = (*RecordsService)(&client.common)
    client.Tools = (*ToolsService)(&client.common)
    return client
}

func InitSSH (sshconfig *SSHConfig) (*SSHBot.Client, bool) {
    sshbot, ErrorMassage := SSHBot.NewSSHConnectionStart(sshconfig.SSHUserName, sshconfig.Address, 22, sshconfig.PathToSSHKey, sshconfig.SSHKeyPass)                    

    if ErrorMassage != nil {
        fmt.Println("Cannot connect to " + sshconfig.Address + " exit with error:\n" + ErrorMassage.Error() + "\n")
        if err := SSHBot.AddAccessDeniedHostToLogFile(sshconfig.PathToAccsessDeniedLogFile, sshconfig.Address, ErrorMassage); err != nil {
            fmt.Println("Error with add new row to file: " + sshconfig.PathToAccsessDeniedLogFile + " exit with error:\n" + err.Error())
        }
        return nil, false                  
    } 
    return sshbot, true
}

func (cmd *RecordsService) PrintA(CIDR string) {
    subnetCIDRs, err := NetworkUtils.GetSubnets(CIDR, 24)
    if err != nil{
        log.Fatal(err)
    }
    for _, sia := range subnetCIDRs{
        if collection, err := cmd.client.Tools.CollectIdenticalRecord("search-data?q=" + sia + "*&object_type=record", "A"); err == nil {
         fmt.Print(collection, "\n\n")
     }
 }
}

func (cmd *ToolsService) CollectIdenticalRecord (SearchRequest string, TypeofCollection string) (map[string][]string, error) {
    SearchData, err := cmd.client.Powerdns.Search(SearchRequest)
    if len(SearchData.([]interface{})) == 0 || err != nil  {
        return nil, err
    }
    records := make(map[string][]string)
    for index := range SearchData.([]interface{}) {
        record := SearchData.([]interface{})[index].(map[string]interface{})
        if (record["type"]).(string) == TypeofCollection {   
            ip := record["content"].(string)
            if _, flag := records[ip]; flag {
                records[ip] = append(records[ip], record["name"].(string))
                continue
            }    
            records[ip] = []string {record["name"].(string)}
        }
    }
    return records, nil  
}

//main function where adds or updates PTR records based on specific algorithm
func (cmd *ToolsService) MakeMeFeelGood (CIDR string, sshconfig *SSHConfig) (err error){
    subnetCIDRs, err := NetworkUtils.GetSubnets(CIDR, 24)
    if err != nil{
        log.Fatal(err)
    }

    for _, sia := range subnetCIDRs{
        // for default_ptr_record_name, index := "unassigned.", 11; index <= 11; default_ptr_record_name, index = "unassigned.", index + 1 {
         for index := 1; index <= 254; index++ {
            ip := sia + "." + strconv.Itoa(index)
            fmt.Println("-------" + ip + "-------")

            //check host reacheble or not
            if flag, ErrorMassage := NetworkUtils.Ping(ip); !flag{
                fmt.Print(ip + " IS UNREACHABLE exit with error: " + ErrorMassage.Error())
                if err := NetworkUtils.AddDeadHostToLogFile(cmd.client.PDNSConfig.PathToDeadHostsLogFile, ip, ErrorMassage); err != nil {
                    fmt.Println("Error with add new row to file: " + cmd.client.PDNSConfig.PathToDeadHostsLogFile + " exit with error:\n" + err.Error())
                }
                cmd.client.Powerdns.PushDNSRecordHostPtr(map[string]interface{}{"name": ip + "." + "unassigned.", "content": ip})
                continue
            }

            sshconfig.Address = ip

            //check mb we alredy have A record for this host
            // add only one first record:
            // cmd.client.Powerdns.PushDNSRecordHostPtr(map[string]interface{}{"name": record.([]interface{})[0].(map[string]interface{})["name"].(string), "content": ip})
            //add all record found:
            if collection, err := cmd.CollectIdenticalRecord("search-data?q=" + ip + "&object_type=record", "A"); collection != nil && err == nil  {
                cmd.client.Powerdns.PushArrayOfDNSRecordHostPtr(collection, 10)
                continue
            }

            //try connect to host use ssh for greb hostaname
            if sshbot, flag := InitSSH(sshconfig); flag {
                output, ErrorMassage := sshbot.RunCommand("hostname -f")
                if ErrorMassage != nil {
                    fmt.Print("Commnad exit with error: " + ErrorMassage.Error())
                    cmd.client.Powerdns.PushDNSRecordHostPtr(map[string]interface{}{"name": ip + "." + "unassigned.", "content": ip})
                    continue
                }
                cmd.client.Powerdns.PushDNSRecordHostPtr(map[string]interface{}{"name": output + ".", "content": ip})
                continue
            } 

            //if cant connet add unassigned PTR record
            cmd.client.Powerdns.PushDNSRecordHostPtr(map[string]interface{}{"name": ip + "." + "unassigned.", "content": ip})
        }    

    }
    return nil
}