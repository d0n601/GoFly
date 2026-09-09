package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"gofly/api"
	"gofly/app"
)

func main() {
	htt := &http.Client{Timeout: 30 * time.Second}
	client := api.NewClient(htt)
	forecast := api.NewOpenMeteoClient(htt)
	a := &app.App{
		Client:   client,
		Forecast: forecast,
		Out:      os.Stdout,
		Now:      time.Now,
	}
	err := a.Run(context.Background(), os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "gofly:", err)
		if errors.Is(err, app.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
