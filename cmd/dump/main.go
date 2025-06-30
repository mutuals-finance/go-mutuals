package dump

import (
	"database/sql"
	"fmt"
	migrate "github.com/mutuals/go-mutuals/db"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"os"
)

func main() {

	username := "user"
	password := "example"
	hostname := "examplehost"
	db := "dbname"
	outputDir := "path/to/example"
	port := 5432

	BackupPostgreSQL(username, password, hostname, db, outputDir, port)
}

func init() {
	viper.SetDefault("POSTGRES_USER", "postgres")
	viper.SetDefault("POSTGRES_PASSWORD", "postgres")
	viper.SetDefault("POSTGRES_DB", "postgres")
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", "")
	viper.AutomaticEnv()
}

func main() {
	migrationsTarget := "./db/migrations/indexer"

	superRequired, err := migrate.SuperUserRequired(migrationsTarget)
	if err != nil {
		panic(err)
	}

	var superClient *sql.DB

	if superRequired {
		var user string
		fmt.Print("Username to use for privileged migrations: ")
		fmt.Scanln(&user)

		fmt.Printf("Password for %s: ", user)
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		fmt.Println("\nAttempting to connect...")

		superClient = postgres.MustCreateClient(
			postgres.WithUser(user),
			postgres.WithPassword(string(pw)),
		)
	}

	if err := dump(superClient, migrations); err != nil {
		fmt.Fprint(os.Stderr, err)
	}
}
