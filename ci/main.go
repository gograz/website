package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dagger.io/dagger"
)

func main() {
	flag.Parse()
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer client.Close()

	switch flag.Arg(0) {
	case "build":
		if err := buildCommand(ctx, client); err != nil {
			log.Fatal(err.Error())
		}
	default:
		os.Exit(1)
	}
}

func buildCommand(ctx context.Context, client *dagger.Client) error {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://gograz.org"
	}

	workdir := client.Host().Workdir()

	container := client.Container().
		From("klakegg/hugo:0.107.0-ext").
		WithMountedDirectory("/src", workdir).
		WithWorkdir("/src").
		WithEnvVariable("HUGO_ENVIRONMENT", "production").
		WithEnvVariable("HUGO_ENV", "production").
		WithExec([]string{
			"--minify",
			"--buildFuture",
			"--baseURL",
			targetURL,
		})

	code, err := container.ExitCode(ctx)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("build exited with code %d", code)
	}

	if _, err := container.Directory("/src/public").Export(ctx, filepath.Join(pwd, "public")); err != nil {
		return err
	}
	return nil
}
