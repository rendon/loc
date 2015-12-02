package main

import (
	"fmt"
	"io/ioutil"
	logger "log"
	"math/rand"
	"os"
	//"sort"
	"strings"
	"time"

	"bitbucket.org/criticalmasser/goapis/results"
	"github.com/codegangsta/cli"
	"gopkg.in/mgo.v2"
)

type User struct {
	ID                 string
	RawLocation        string
	NormalizedLocation *Location
}

type FrequencyItem struct {
	Code     string
	Quantity int
}

type Frequency struct {
	Items []FrequencyItem
}

func init() {
	rand.Seed(time.Now().Unix())
}

func (f Frequency) Len() int {
	return len(f.Items)
}

func (f Frequency) Less(i, j int) bool {
	return f.Items[i].Quantity < f.Items[j].Quantity
}

func (f Frequency) Swap(i, j int) {
	t := f.Items[i]
	f.Items[i] = f.Items[j]
	f.Items[j] = t
}

var (
	log *logger.Logger
)

func randColor() string {
	r := rand.Int() % 256
	g := rand.Int() % 256
	b := rand.Int() % 256
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func genDatamapsFile(users []User) {
	/*
		freq := Frequency{
			Items: make([]FrequencyItem, 0),
		}
		for k, v := range f {
			freq.Items = append(freq.Items, FrequencyItem{k, v})
		}
		sort.Sort(freq)
		for _, item := range freq.Items {
			fmt.Printf("%q: %q,\n", item.Code, randColor())
		}
		fmt.Println()
		for _, item := range freq.Items {
			fmt.Printf("%q: { fillKey: %q },\n", item.Code, item.Code)
		}
		fmt.Println()
		for _, item := range freq.Items {
			fmt.Printf("%q: %d,\n", item.Code, item.Quantity)
		}

		for i := 0; i < len(mx); i++ {
			fmt.Printf("%s ", mx[i].ID)
		}
	*/
}

func loc(c *cli.Context) {
	users := make([]User, 0)
	if c.String("file") != "" {
		buf, err := ioutil.ReadFile(c.String("file"))
		if err != nil {
			log.Fatal("Failed to read input file: %s", err)
		}
		for _, line := range strings.Split(string(buf), "\n") {
			if line != "" {
				u := User{RawLocation: line}
				users = append(users, u)
			}
		}
	} else {
		if len(c.Args()) != 1 {
			log.Fatal("USAGE: ./main <dbname>")
		}
		log.Printf("Getting locations from server...")
		dbname := c.Args()[0]
		session, err := mgo.Dial("mongodb-server")
		if err != nil {
			log.Fatal(err)
		}

		col := session.DB(dbname).C("graph")
		nodes := col.Find(nil).Iter()
		var r results.Node
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
	}
	log.Printf("Initializing database...")
	initialize()
	log.Printf("users: %d\n", len(users))

	for i := 0; i < len(users); i++ {
		u := &users[i]
		if u.RawLocation == "" {
			continue
		}
		l := normalizeLocation(u.RawLocation)
		if l != nil {
			u.NormalizedLocation = l
		}
	}

	for _, u := range users {
		if u.NormalizedLocation != nil {
			country := u.NormalizedLocation.Country
			fmt.Printf("%q,%q,%q\n", u.ID, u.RawLocation, country)
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "loc"
	app.Version = "0.1.0"
	app.Usage = "Normalize user locations"
	app.ArgsUsage = "<dbname>"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file",
			Usage: "Analize file",
		},
	}
	app.Action = loc
	app.Run(os.Args)
}
