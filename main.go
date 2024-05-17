package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	homeDir, _    = os.UserHomeDir()
	appDir        = filepath.Join(homeDir, ".config/glab-ctx")
	glabDir       = filepath.Join(homeDir, ".config/glab-cli")
	usersFilename = filepath.Join(appDir, "users.yml")
	glabFilename  = filepath.Join(glabDir, "config.yml")
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout})

	if len(os.Args) != 3 {
		err := errors.New("missing argument")
		log.Fatal().Err(err).Msg("./app <host> <ctx>")
	}
	os.MkdirAll(appDir, 0644)
	os.MkdirAll(glabDir, 0644)

	host := os.Args[1]
	name := os.Args[2]

	userFS, err := os.Open(usersFilename)
	if err != nil {
		log.Fatal().Err(err).Msg("open users config file")
	}
	defer userFS.Close()

	var config Config
	if err := yaml.NewDecoder(userFS).Decode(&config); err != nil {
		log.Fatal().Err(err).Msg("parse users config file")
	}
	contexts, exists := config.Hosts[host]
	if !exists {
		err := fmt.Errorf("host %s not found", host)
		log.Fatal().Err(err).Msg("select host")
	}
	var hostCtx Context
	for _, ctx := range contexts.Contexts {
		if name == ctx.Name {
			hostCtx = ctx
			break
		}
	}
	if !hostCtx.IsValid() {
		err := errors.New("invalid host context")
		log.Fatal().Err(err).Msg("select host context")
	}

	configFS, err := os.Open(glabFilename)
	if err != nil {
		log.Fatal().Err(err).Msg("open glab config file")
	}
	defer configFS.Close()

	var content map[string]any
	if err := yaml.NewDecoder(configFS).Decode(&content); err != nil {
		log.Fatal().Err(err).Msg("parse glab config file")
	}
	if _, exists := content["hosts"]; !exists {
		log.Fatal().Err(err).Msg("config hosts not found")
	}
	hosts, ok := content["hosts"].(map[string]any)
	if !ok {
		log.Fatal().Err(err).Msg("config hosts not found")
	}
	if _, exists := hosts[contexts.Host]; !exists {
		log.Fatal().Err(err).Msg("config hosts not found")
	}
	ctx := hosts[contexts.Host].(map[string]any)
	ctx["user"] = hostCtx.User
	ctx["token"] = hostCtx.Token
	hosts[contexts.Host] = ctx
	content["hosts"] = hosts
	configFS.Close()

	configFS, err = os.Create(glabFilename)
	if err != nil {
		log.Fatal().Err(err).Msg("write glab config to file")
	}
	defer configFS.Close()

	if err := yaml.NewEncoder(configFS).Encode(content); err != nil {
		log.Fatal().Err(err).Msg("write new config to file")
	}
}

type Config struct {
	Hosts map[string]Host `yaml:"hosts"`
}

type Host struct {
	Host     string    `yaml:"host"`
	Contexts []Context `yaml:"contexts"`
}

type Context struct {
	Name  string `yaml:"name"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
}

func (c Context) IsValid() bool {
	return c.Name != "" && c.User != "" && c.Token != ""
}
