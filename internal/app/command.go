package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/zainul/gan/internal/app/io"

	"github.com/zainul/gan/internal/app/constant"
)

// Migration is type for creating thing that related with migration database
type MigrationCommand interface {
	CreateFile(name string, extention string, fileType string) error
	Migrate(status string)
	Seed()
}

type storeMigration struct {
	// dir is directory for migrations file will be placed
	Dir string `json:"dir"`
	// conn is connection string to DB
	Conn string `json:"conn"`
	// seedDir  is directory for seed file will be placed
	SeedDir string `json:"seed_dir"`
}

// NewMigration ..
func NewMigration(dir string, conn string, seedDir string) MigrationCommand {
	os.Setenv(constant.CONNDB, conn)
	os.Setenv(constant.DIR, dir)
	return &storeMigration{dir, conn, seedDir}
}

func (s *storeMigration) Seed() {
	// 1. make build by cmd
	// 2. run the binary
	// 3. delete the binary
	ganseed := "ganseed"

	// 1. make build by cmd
	changeDirectory(s.SeedDir)
	cmd := exec.Command("go", "build", "-o", ganseed)

	if _, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("read binary error while seed ", err)
		deleteTempFile(s.SeedDir, ganseed)
		os.Exit(2)
	}

	fmt.Println("step 1. make build by cmd done ...")

	// 2. run the binary
	cmd = exec.Command(fmt.Sprintf("./%v", ganseed))
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("error while run binary ", err, string(out))
		deleteTempFile(s.SeedDir, ganseed)
		os.Exit(2)
	} else {
		fmt.Println("========================================================")
		fmt.Println("SEEDER START")
		fmt.Println("========================================================")
		fmt.Println(string(out))
		fmt.Println("========================================================")
	}
	fmt.Println("step 2. run the binary done...")

	// 3. delete the binary
	deleteTempFile(s.SeedDir, ganseed)
	fmt.Println("step 3. delete the binary done...")
	os.Exit(2)
}

func (s *storeMigration) Migrate(status string) {
	// 1. create migrate by up or down in selected location
	// 2. make build by cmd
	// 3. run the binary
	// 4. delete the binary and main.go

	// implementation
	// 1. create migrate by up or down in selected location
	if status == constant.StatusUp {
		s.CreateFile("main", constant.DotGo, constant.FileTypeMigrationUp)
	} else if status == constant.StatusDown {
		s.CreateFile("main", constant.DotGo, constant.FileTypeMigrationUp)
	}

	// 2. make build by cmd
	changeDirectory(s.Dir)
	cmd := exec.Command("go", "build", "-o", "ganrun")

	if _, err := cmd.CombinedOutput(); err != nil {
		// TODO: make remove temp binary
		deleteTempFile(s.Dir, "main.go")
		deleteTempFile(s.Dir, "ganrun")
		os.Exit(2)
	}

	// 3. run the binary
	changeDirectory(s.Dir)
	cmd = exec.Command("./ganrun")

	if out, err := cmd.CombinedOutput(); err != nil {
		// TODO: make remove temp binary
		deleteTempFile(s.Dir, "main.go")
		deleteTempFile(s.Dir, "ganrun")
		os.Exit(2)
	} else {
		fmt.Println("========================================================")
		fmt.Println("MIGRATION START")
		fmt.Println("========================================================")
		fmt.Println(string(out))
		fmt.Println("========================================================")
	}

	// 4. delete the binary and main.go
	deleteTempFile(s.Dir, "main.go")
	deleteTempFile(s.Dir, "ganrun")
	fmt.Println("Gan Migration success ...  !!!")
	os.Exit(2)

}

func changeDirectory(dir string) {
	if err := os.Chdir(dir); err != nil {
		fmt.Println("Could not find migration directory: %s", err)
	}
}

func deleteTempFile(dir, file string) {
	changeDirectory(dir)
	if err := os.Remove(file); err != nil {
		fmt.Println("Could not remove temporary file: %s", err)
	}
}

func (s *storeMigration) CreateFile(name string, extention string, fileType string) error {

	AppPath := fmt.Sprintf("%v/%v", os.Getenv("GOPATH"), constant.PathAppName)

	sourceFilename := fmt.Sprintf("%v/internal/app/templates/%v.tpl", AppPath, fileType)

	destinationFilename := fmt.Sprintf("%v/%v.%v", s.Dir, name, extention)

	if fileType == constant.FileTypeCreationSeed {
		destinationFilename = fmt.Sprintf("%v/%v.%v", s.SeedDir, name, extention)
	}

	// detect if file exists
	var _, err = os.Stat(destinationFilename)
	fmt.Println("will be create file in directory ...", destinationFilename)

	// create file if not exists
	if os.IsNotExist(err) {
		tmpl, err := template.ParseFiles(sourceFilename)

		if err != nil {
			fmt.Println("Failed create file ...", err)
			return err
		}

		data := struct {
			Key          string
			KeyLowerCase string
		}{
			Key:          strings.Title(name),
			KeyLowerCase: strings.ToLower(name),
		}

		var tpl bytes.Buffer
		err = tmpl.Execute(&tpl, data)

		if err != nil {
			fmt.Println("failed creating the file ", err)
		}

		tplStr := tpl.String()
		io.WriteFile(destinationFilename, tplStr)
	}

	fmt.Println("done creating file ")
	return nil
}
