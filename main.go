// package main
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tmdbapi "earapi/tmdb"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var (
	apiversion = "v0.0.26"
	configFile string
	ip         string
	port       string
	config     tmdbapi.EarapiSettings
)

var rootCmd = &cobra.Command{
	Use:     "earapi",
	Short:   "Start the earapi server",
	Version: apiversion,
	Run: func(cmd *cobra.Command, args []string) {
		loadConfig()

		if cmd.Flags().Changed("ip") {
			config.API.IP = ip
		} else if config.API.IP == "" {
			config.API.IP = ip
		}

		if cmd.Flags().Changed("port") {
			config.API.Port = port
		} else if config.API.Port == "" {
			config.API.Port = port
		}

		r := gin.Default()

		// Handler for the root path
		r.GET("/", func(c *gin.Context) { rootHandler(c, r) })

		steamv1Group := r.Group("/steam/v1/")
		{
			steamv1Group.GET("/top", steamTopHandler)
			steamv1Group.GET("/getuserid", steamUserIDHandler)
			steamv1Group.GET("/appsused", steamUserAppsUsedHandler)
			steamv1Group.GET("/appdata", steamAppDataHandler)
			steamv1Group.GET("/search", searchSteamAppHandler)
		}

		r.GET("/joke", jokeHandler)

		tmdbGroup := r.Group("/tmdb/v1/")
		{
			tmdbGroup.GET("/search", tmdbapi.SearchHandler(&config))
		}

		netflixGroup := r.Group("/netflix/v1/")
		{
			netflixGroup.GET("/top", netflixTopHandler)
		}

		r.GET("/version", versionHandler)

		httpserver := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", config.API.IP, config.API.Port),
			Handler: r,
		}

		go func() {
			if err := httpserver.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Println(err)
			}
		}()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

		<-signals
		fmt.Println("Shutting down the API")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpserver.Shutdown(ctx); err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "config/earapi.json", "Path to config file")
	rootCmd.Flags().StringVar(&ip, "ip", "127.0.0.1", "IP address to listen on")
	rootCmd.Flags().StringVar(&port, "port", "8080", "Port to listen on")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": apiversion,
	})
}

func rootHandler(c *gin.Context, r *gin.Engine) {
	routes := r.Routes()
	var endpoints []string
	for _, route := range routes {
		endpoints = append(endpoints, fmt.Sprintf("%s - %s", route.Method, route.Path))
	}
	endpoints = append(endpoints, fmt.Sprintf("%s - %s", "GET", "/doc"))
	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
	})
}
