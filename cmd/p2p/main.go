package main

import (
	"flag"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/nguyenkhoa0721/libp2p-wrapper/config"
	"github.com/nguyenkhoa0721/libp2p-wrapper/internal/p2p"
	_ "github.com/nguyenkhoa0721/libp2p-wrapper/pkg/log"
	"github.com/sirupsen/logrus"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "./config.yml", "Path to config.yml")
}

func main() {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	net, err := p2p.NewNet(cfg.P2P.Addr, cfg.P2P.Secret)
	if err != nil {
		panic(err)
	}

	net.Host.SetStreamHandler("/", func(s network.Stream) {
		mesage, err := p2p.HandleStream(s)
		if err != nil {
			logrus.Error(err)
			return
		}

		logrus.Info(mesage)
	})

	for _, dest := range cfg.P2P.Dests {
		func(dest string) {
			peerId := net.ConnectToPeer(dest)
			logrus.Infof("Connected to %s with peerId %s", dest, peerId)
		}(dest)
	}
}
