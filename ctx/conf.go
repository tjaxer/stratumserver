package ctx

import (
	conf "github.com/nixys/nxs-go-conf"
)

type confOpts struct {
	Bind              string    `conf:"bind" conf_extraopts:"default=0.0.0.0:8080"`
	LogFile           string    `conf:"logfile" conf_extraopts:"default=stdout"`
	LogLevel          string    `conf:"loglevel" conf_extraopts:"default=info"`
	PidFile           string    `conf:"pidfile"`
	ClientMaxBodySize string    `conf:"clientMaxBodySize" conf_extraopts:"default=36m"`
	TLS               tlsConf   `conf:"tls"`
	//AuthKey           string    `conf:"authKey" conf_extraopts:"required"`
}

type tlsConf struct {
	CertFile string `conf:"certfile"`
	KeyFie   string `conf:"keyfile"`
}


func confRead(confPath string) (confOpts, error) {

	var c confOpts

	err := conf.Load(&c, conf.Settings{
		ConfPath:    confPath,
		ConfType:    conf.ConfigTypeYAML,
		UnknownDeny: true,
	})
	if err != nil {
		return c, err
	}


	return c, err
}
