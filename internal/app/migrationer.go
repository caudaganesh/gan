package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/zainul/gan/internal/app/constant"
	"github.com/zainul/gan/internal/app/database"
	"github.com/zainul/gan/internal/app/io"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Migrationer interface {
	Up()
	Down()
	Exec(status string)
	GetSQL() string
}

var (
	migrationMap map[string]Migrationer
)

func init() {
	migrationMap = make(map[string]Migrationer)
}

type Migration struct {
	sql          string
	unixNanoTime float64
	key          string
}
type Store interface {
	Create(v interface{}) error
}

func GetDB() *gorm.DB {
	db, err := gorm.Open("postgres", os.Getenv(constant.CONNDB))
	if err != nil {
		fmt.Println("failed to get instance")
		return nil
	}

	return db
}

// Seed ...
func Seed(path string, store Store, value ...interface{}) {
	byteData, err := io.OpenFile(path)

	if err != nil {
		fmt.Println("error before seed ", err)
		return
	}
	err = json.Unmarshal(byteData, &value)

	if err != nil {
		fmt.Println("error unmarshal type ", err)
	}

	if constant.CONNDB == "" {
		fmt.Println("please configure connection first ...")
		return
	}

	if err != nil {
		fmt.Println("please fill with correct connection string ", err)
	}

	fmt.Println("seed data from file ", path, " start ...")

	for _, val := range value {

		err = store.Create(val)

		if err != nil {
			fmt.Println("error while created data ", err)
			return
		}
	}
	fmt.Println("seed data from file ", path, " complete ...")

	return
}

func Register(name string, m Migrationer) error {
	if _, ok := migrationMap[name]; ok {
		return errors.New("already exist name:" + name)
	}
	migrationMap[name] = m
	return nil
}

func (m *Migration) Exec(status string) {
	if status == constant.StatusUp {
		migrations := make([]Migration, 0)

		for key, val := range migrationMap {
			val.Up()
			timeUnix, err := splitterTimeFromKey(key)

			if err != nil {
				fmt.Println("migration failed please fix your file ", err)
				return
			}
			migrations = append(migrations, Migration{
				key:          key,
				sql:          val.GetSQL(),
				unixNanoTime: timeUnix,
			})
		}

		sort.SliceStable(migrations, func(i, j int) bool {
			return migrations[i].unixNanoTime < migrations[j].unixNanoTime
		})

		// fmt.Println(migrations)

		if constant.CONNDB == "" {
			fmt.Println("please configure connection first ...")
			return
		}

		conn, err := sql.Open("postgres", os.Getenv(constant.CONNDB))

		if err != nil {
			fmt.Println("failed make connection to DB please configure right connection")
			return
		}
		db := database.NewDB(conn)

		err = db.Exec(constant.MigrationTablePG)

		if err != nil {
			fmt.Println("Failed craete migrations table ", err)
		}

		for _, val := range migrations {

			err = db.GetByMigrationKey(val.key)

			if err != nil {
				fmt.Println("get migration by key => ", err)
				continue
			}

			err = db.Exec(val.sql)
			now := time.Now()
			sch := database.Schema{
				Up:        true,
				ExecuteUp: &now,
				Migration: val.key,
				Statement: val.sql,
			}

			if err != nil {
				fmt.Println("Failed to execute the migration ", err)
				os.Exit(2)
			}

			if err = db.Save(sch); err != nil {
				fmt.Println("error when create history migrations  ", err)
			}
		}

		// fmt.Println(migrations)
	} else if status == constant.StatusDown {
		fmt.Println("status down", m.sql)
	}
}

func splitterTimeFromKey(key string) (float64, error) {
	strArr := strings.Split(key, "_")
	latestStr := strArr[len(strArr)-1]

	return strconv.ParseFloat(latestStr, 64)
}

func (m *Migration) SQL(sql string) {
	m.sql = sql
}

func (m *Migration) SQLFromFile(path string) {
	byteData, err := io.OpenFile(path)

	if err != nil {
		fmt.Println("Failed to open file ", err)
		return
	}

	m.sql = string(byteData)
}

func (m *Migration) GetSQL() string {
	return m.sql
}
