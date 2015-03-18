package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	defaultRouteChoice = "ordered"
	defaultSshExe      = "ssh"
	defaultSshPort     = "22"
	defaultSshArgs     = []string{"-q", "-Y"}
)

type duration time.Duration

func (d *duration) UnmarshalText(text []byte) error {
	td, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = duration(td)
	return nil
}

type sshProxyConfig struct {
	Debug          bool
	Log            string
	Dump           string
	Stats_Interval duration
	Bg_Command     string
	Route_Choice   string
	Ssh            sshConfig
	Environment    map[string]string
	Routes         map[string][]string
	Users          map[string]subConfig
	Groups         map[string]subConfig
}

type sshConfig struct {
	Exe  string
	Args []string
}

type subConfig struct {
	Debug          bool
	Log            string
	Dump           string
	Stats_Interval duration
	Bg_Command     string
	Route_Choice   string
	Environment    map[string]string
	Routes         map[string][]string
	Ssh            sshConfig
}

func parseSubConfig(md *toml.MetaData, config *sshProxyConfig, subconfig *subConfig, subgroup, subname string) {
	if md.IsDefined(subgroup, subname, "debug") {
		config.Debug = subconfig.Debug
	}

	if md.IsDefined(subgroup, subname, "log") {
		config.Log = subconfig.Log
	}

	if md.IsDefined(subgroup, subname, "dump") {
		config.Dump = subconfig.Dump
	}

	if md.IsDefined(subgroup, subname, "bg_command") {
		config.Bg_Command = subconfig.Bg_Command
	}

	if md.IsDefined(subgroup, subname, "route_choice") {
		config.Route_Choice = subconfig.Route_Choice
	}

	if md.IsDefined(subgroup, subname, "ssh", "exe") {
		config.Ssh.Exe = subconfig.Ssh.Exe
	}

	if md.IsDefined(subgroup, subname, "ssh", "args") {
		config.Ssh.Args = subconfig.Ssh.Args
	}

	if md.IsDefined(subgroup, subname, "routes") {
		config.Routes = subconfig.Routes
	}

	if md.IsDefined(subgroup, subname, "environment") {
		// merge environment
		for k, v := range subconfig.Environment {
			config.Environment[k] = v
		}
	}
}

type PatternReplacer struct {
	Regexp *regexp.Regexp
	Text   string
}

func replace(src string, replacer *PatternReplacer) string {
	return replacer.Regexp.ReplaceAllString(src, replacer.Text)
}

func loadConfig(config_file, username, sid string, start time.Time, groups map[string]bool) (*sshProxyConfig, error) {
	patterns := map[string]*PatternReplacer{
		"{user}": &PatternReplacer{regexp.MustCompile(`{user}`), username},
		"{sid}":  &PatternReplacer{regexp.MustCompile(`{sid}`), sid},
		"{time}": &PatternReplacer{regexp.MustCompile(`{time}`), start.Format(time.RFC3339Nano)},
	}

	var config sshProxyConfig
	md, err := toml.DecodeFile(config_file, &config)
	if err != nil {
		return nil, err
	}

	if !md.IsDefined("routes") {
		return nil, fmt.Errorf("no routes specified")
	}

	if !md.IsDefined("route_choice") {
		config.Route_Choice = defaultRouteChoice
	}

	if !md.IsDefined("ssh", "exe") {
		config.Ssh.Exe = defaultSshExe
	}

	if !md.IsDefined("ssh", "args") {
		config.Ssh.Args = defaultSshArgs
	}

	for _, key := range md.Keys() {
		if key[0] == "groups" {
			groupname := key[1]
			if groups[groupname] {
				groupconfig := config.Groups[groupname]
				parseSubConfig(&md, &config, &groupconfig, "groups", groupname)
			}
		}
	}

	if userconfig, present := config.Users[username]; present {
		parseSubConfig(&md, &config, &userconfig, "users", username)
	}

	if config.Log != "" {
		config.Log = replace(config.Log, patterns["{user}"])
	}

	for k, v := range config.Environment {
		config.Environment[k] = replace(v, patterns["{user}"])
	}

	if _, ok := routeChoosers[config.Route_Choice]; !ok {
		return nil, fmt.Errorf("invalid value for `route_choice` option: %s", config.Route_Choice)
	}

	if config.Dump != "" {
		for _, repl := range patterns {
			config.Dump = replace(config.Dump, repl)
		}
	}

	return &config, nil
}
