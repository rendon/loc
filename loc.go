package main

import (
	"fmt"
	logger "log"
	"os"

	"github.com/codegangsta/cli"
	"gopkg.in/mgo.v2"
)

var (
	log *logger.Logger
)

func init() {
	log = logger.New(os.Stderr, "", 0)
}

func loc(c *cli.Context) {
	if len(c.Args()) != 1 {
		log.Fatal("USAGE: ./main <dbname>")
	}
	if c.Bool("strict") {
		fmt.Printf("Strict mode!\n")
	}
	dbname := c.Args()[0]
	session, err := mgo.Dial("mongodb-server")
	if err != nil {
		log.Fatal(err)
	}

	db := session.DB(dbname)
	names, err := db.CollectionNames()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("DB name: %s\n", dbname)
	fmt.Printf("%v\n", names)
}

func main() {
	app := cli.NewApp()
	app.Name = "loc"
	app.Version = "0.1.0"
	app.Usage = "Normalize user locations"
	app.ArgsUsage = "<dbname>"
	app.Action = loc
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "strict",
			Usage: "Only work with explored users",
		},
	}
	app.Run(os.Args)
}
