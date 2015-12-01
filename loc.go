package main

import (
	"fmt"
	logger "log"
	"math/rand"
	"os"
	"sort"
	"time"

	"bitbucket.org/criticalmasser/goapis/results"
	"github.com/codegangsta/cli"
	"gopkg.in/mgo.v2"
)

type User struct {
	ID                 string
	RawLocation        string
	NormalizedLocation string
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

func loc(c *cli.Context) {
	if len(c.Args()) != 1 {
		log.Fatal("USAGE: ./main <dbname>")
	}
	log.Printf("Initializing database...")
	initialize()

	log.Printf("Getting locations from server...")
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
	log.Printf("users: %d\n", len(users))

	var f = make(map[string]int)
	mx := make([]User, 0)
	for i := 0; i < len(users); i++ {
		u := &users[i]
		if u.RawLocation == "" {
			continue
		}
		l := normalizeLocation(u.RawLocation)
		if l != nil {
			//if l.Address != "" {
			//    fmt.Printf("%s:%q:%v\n", u.ID, u.RawLocation, l)
			//}
			f[l.LongCountryCode]++
			if l.ShortCountryCode == "MX" {
				mx = append(mx, *u)
			}
		} else {
			//fmt.Printf("&{%q, %q}\n", u.ID, u.RawLocation)
		}
	}

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
