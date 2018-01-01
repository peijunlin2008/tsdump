package main

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"github.com/urfave/cli"
	"github.com/voidint/tsdump/build"
	"github.com/voidint/tsdump/config"
	"github.com/voidint/tsdump/model"
	"github.com/voidint/tsdump/model/mysql"
	"github.com/voidint/tsdump/view"
	"github.com/voidint/tsdump/view/txt"
	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/voidint/tsdump/view/csv"
	_ "github.com/voidint/tsdump/view/json"
	_ "github.com/voidint/tsdump/view/md"
	_ "github.com/voidint/tsdump/view/txt"
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

var (
	c   config.Config
	out io.Writer = os.Stdout
)

func main() {
	cli.HelpFlag = cli.BoolFlag{
		Name:  "help",
		Usage: "show help",
	}

	app := cli.NewApp()
	app.Name = "tsdump"
	app.Usage = "Database table structure dump tool."
	app.Version = build.Version("0.2.0")
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "voidnt",
			Email: "voidint@126.com",
		},
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "D, debug",
			Usage:       "enable debug mode",
			Destination: &c.Debug,
		},
		cli.StringFlag{
			Name:        "h, host",
			Value:       "127.0.0.1",
			Usage:       "connect to host",
			Destination: &c.Host,
		},
		cli.IntFlag{
			Name:        "P, port",
			Value:       3306,
			Usage:       "port number to use for connection",
			Destination: &c.Port,
		},
		cli.StringFlag{
			Name:        "u, user",
			Value:       username,
			Usage:       "user for login if not current user",
			Destination: &c.Username,
		},
		cli.StringFlag{
			Name:  "V, viewer",
			Value: txt.Name,
			Usage: fmt.Sprintf(
				"output viewer. Optional values: %s",
				strings.Join(view.Registered(), "|"),
			),
			Destination: &c.Viewer,
		},
		cli.StringFlag{
			Name:        "o, output",
			Usage:       "write to a file, instead of STDOUT",
			Destination: &c.Output,
		},
	}

	app.Before = func(ctx *cli.Context) (err error) {
		if args := ctx.Args(); len(args) > 0 {
			c.DB = args.First()
			c.Tables = args.Tail()
		}

		var passwd []byte
		if passwd, err = readPassword(); err != nil {
			return cli.NewExitError(fmt.Sprintf("[tsdump] %s", err.Error()), 1)
		}
		c.Password = string(passwd)
		return nil
	}

	app.Action = func(ctx *cli.Context) (err error) {
		repo, err := mysql.NewRepo(&c)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("[tsdump] %s", err.Error()), 1)
		}

		// Get db and table metadata
		dbs, err := getMetadata(repo, c.DB, c.Tables...)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("[tsdump] %s", err.Error()), 1)
		}

		if c.Output != "" {
			var f *os.File
			if f, err = os.Create(c.Output); err != nil {
				return cli.NewExitError(fmt.Sprintf("[tsdump] %s", err.Error()), 1)
			}
			defer f.Close()
			out = f
		}

		// Output as target viewer
		v := view.SelectViewer(c.Viewer)
		if v == nil {
			return cli.NewExitError(fmt.Sprintf("[tsdump] unsupported viewer: %q", c.Viewer), 1)
		}
		if err = v.Do(dbs, out); err != nil {
			return cli.NewExitError(fmt.Sprintf("[tsdump] %s", err.Error()), 1)
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("[tsdump] %s", err.Error()))
		os.Exit(1)
	}
}

// readPassword 从stdin读取密码
func readPassword() (passwd []byte, err error) {
	defer fmt.Println()
	fmt.Print("Enter Password: ")
	return terminal.ReadPassword(int(os.Stdin.Fd()))
}

// getMetadata 根据目标数据库名和表名，返回目标数据库及其表的元数据。
func getMetadata(repo model.IRepo, db string, tables ...string) (dbs []model.DB, err error) {
	if db == "" && len(tables) > 0 {
		panic("unreachable")
	}

	// 获取所有数据库下的表
	if db == "" {
		return repo.GetDBs(nil, false)
	}

	// 获取单个数据库下的表
	if len(tables) == 0 {
		return repo.GetDBs(&model.DB{
			Name: db,
		}, false)
	}

	// 获取单个数据库下的若干表
	dbs, err = repo.GetDBs(&model.DB{
		Name: c.DB,
	}, true)
	if err != nil {
		return nil, err
	}

	for i := range dbs {
		for j := range tables {
			tables, err := repo.GetTables(&model.Table{
				DB:   dbs[i].Name,
				Name: tables[j],
			})
			if err != nil {
				return nil, err
			}
			dbs[i].Tables = append(dbs[i].Tables, tables...)
		}
	}
	return dbs, nil
}
