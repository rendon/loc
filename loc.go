package main

import (
	"fmt"
	logger "log"
	"os"

	"bitbucket.org/criticalmasser/goapis/results"
	"github.com/codegangsta/cli"
	"gopkg.in/mgo.v2"
)

type User struct {
	ID                 string
	RawLocation        string
	NormalizedLocation string
}

var (
	log *logger.Logger
)

func loc(c *cli.Context) {
	if len(c.Args()) != 1 {
		log.Fatal("USAGE: ./main <dbname>")
	}
	dbname := c.Args()[0]
	session, err := mgo.Dial("mongodb-server")
	if err != nil {
		log.Fatal(err)
	}

	col := session.DB(dbname).C("graph")
	nodes := col.Find(nil).Iter()
	var r results.Node
	users := make([]User, 0)
	for nodes.Next(&r) {
		loc, ok := r.Properties["location"].(string)
		if !ok {
			continue
		}
		u := User{
			ID:          r.Start,
			RawLocation: loc,
		}
		users = append(users, u)
	}

	for i := 0; i < len(users); i++ {
		u := &users[i]
		if u.RawLocation == "" {
			continue
		}
		l := normalizeLocation(u.RawLocation)
		if l != nil {
			fmt.Printf("%s:%q:%v\n", u.ID, u.RawLocation, l)
		} else {
			fmt.Printf("&{%q, %q}\n", u.ID, u.RawLocation)
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "loc"
	app.Version = "0.1.0"
	app.Usage = "Normalize user locations"
	app.ArgsUsage = "<dbname>"
	app.Action = loc
	app.Run(os.Args)
}
