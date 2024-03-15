package FixDNSConfig

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
    SSH struct {
        User                       string   `yaml:"User"`
        PSKey                      string   `yaml:"PSKey"`
        PathToKey                  string   `yaml:"PathToKey"`
        PathToAccsessDeniedLogFile string   `yaml:"PathToAccsessDeniedLogFile"`
    } `yaml:"SSH"`
    PDNS struct {
        Host                       string   `yaml:"Host"`
        ApiKey                     string   `yaml:"ApiKey"`
        NameServers                []string `yaml:"NameServers"`
        PathToDeadHostsLogFile     string   `yaml:"PathToDeadHostsLogFile"`
    } `yaml:"PDNS"`
} 

func ReadConfig(configPath string) (*Config, error) {
    
    if err := ValidateConfigPath(configPath); err != nil {
        return nil, err
    }

    config := &Config{}

    // Open YAML file
    file, err := os.Open(configPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Decode YAML file to struct
    if file != nil {
        decoder := yaml.NewDecoder(file) 
        if err := decoder.Decode(&config); err != nil {
            return nil, err
        }
    return config, nil
    }

    return nil, errors.New("Config file is empty!")
}

// ValidateConfigPath just makes sure, that the path provided is a file,that can be read
func ValidateConfigPath(path string) error {
    s, err := os.Stat(path)
    if err != nil {
        return err
    }
    if s.IsDir() {
        return fmt.Errorf("'%s' is a directory, not a normal file", path)
    }
    return nil
}