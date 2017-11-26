package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/urfave/cli"
	"github.com/voidint/tsdump/build"
	"github.com/voidint/tsdump/config"
	"github.com/voidint/tsdump/model"
	"github.com/voidint/tsdump/model/mysql"
	"github.com/voidint/tsdump/view/txt"
)

var (
	username string
)

func init() {
	u, err := user.Current()
	if err == nil {
		username = u.Username
	}
}

var c config.Config

func main() {
	app := cli.NewApp()
	app.Name = ""
	app.Usage = ""
	app.Version = build.Version("0.1.0")

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "H, host",
			Value:       "127.0.0.1",
			Usage:       "Connect to host.",
			Destination: &c.Host,
		},
		cli.IntFlag{
			Name:        "P, port",
			Value:       3306,
			Usage:       "Port number to use for connection.",
			Destination: &c.Port,
		},
		cli.StringFlag{
			Name:        "u, user",
			Value:       username,
			Usage:       "User for login if not current user.",
			Destination: &c.Username,
		},
		cli.StringFlag{
			Name:        "p, password",
			Usage:       "Password to use when connecting to server.",
			Destination: &c.Password,
		},
		cli.StringFlag{
			Name:        "d, db",
			Usage:       "Database name.",
			Destination: &c.DB,
		},
		cli.StringFlag{
			Name:        "V, viewer",
			Value:       "txt",
			Usage:       "Viewer",
			Destination: &c.Viewer,
		},
		cli.StringFlag{
			Name:        "o, output",
			Usage:       "Write to a file, instead of STDOUT.",
			Destination: &c.Output,
		},
		cli.BoolFlag{
			Name:        "D, debug",
			Usage:       "Enable debug mode.",
			Destination: &c.Debug,
		},
	}
	app.Action = func(ctx *cli.Context) error {
		if c.Debug {
			fmt.Println(c)
		}

		repo, err := mysql.NewRepo(&c)
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		// 获取数据
		var dbs []model.DB
		if c.DB != "" {
			dbs, err = repo.GetDBs(&model.DB{
				Name: c.DB,
			})
		} else {
			dbs, err = repo.GetDBs(nil)
		}
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		// 输出到目标
		_ = txt.NewView().Do(dbs, os.Stdout)
		return nil
	}

	app.Run(os.Args)
}