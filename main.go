package main

import (
	_ "app/jobs"
	"app/lib"
	_ "app/migrations"
	"embed"
	"os"

	"github.com/joho/godotenv"
)

//go:embed views assets migrations
var FS embed.FS

func main() {
	if !lib.IsProduction() && os.Getenv("SECRET") == "" {
		os.Setenv("SECRET", "5495ba0bb4c626bc5d9c230db75cff8936230cee8fa882987abcd40c4dd179b0")
	}
	godotenv.Overload()
	lib.SecretsLoad(os.Getenv("SECRET"), secrets[lib.Env("ENV", "development")])
	s := lib.NewServer(FS)
	s.ChainClients[42161] = lib.NewChainClient(42161)
	setupRoutes(s)
	s.Queue.RunCliJob()
}
