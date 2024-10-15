package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/disgoorg/disgo/webhook"

	"catbox-scraper/catbox"
)

func main() {
	err := catbox.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	db, err := catbox.InitDB(catbox.G_config.Database)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	if catbox.G_config.UseProxies {
		var err error
		catbox.G_proxyManager, err = catbox.InitProxyManager(catbox.G_config.ProxyFile)
		if err != nil {
			log.Fatalf("Error initializing proxy manager: %v", err)
		}
	}

	if catbox.G_config.WebhookEnabled {
		client, err := webhook.NewWithURL(catbox.G_config.WebhookURL)
		if err != nil {
			catbox.G_logger.Errorln(err)
			os.Exit(1)
		}
		catbox.G_webhook_client = client
	}

	catbox.EnsureDownloadPathExists(catbox.G_config.DownloadPath)

	var wg sync.WaitGroup
	idChan := make(chan string, catbox.G_config.Workers)

	catbox.G_Req_Per_Sec.Store(0)
	catbox.G_Found_Per_Min.Store(0)

	go func() {
		sec := time.NewTicker(time.Second)
		min := time.NewTicker(time.Minute)
		defer sec.Stop()
		defer min.Stop()
		var rpm int64 = 0
		var total_found int64 = 0
		var total_req int64 = 0
		for {
			select {
			case <-sec.C:
				rps := catbox.G_Req_Per_Sec.Load()
				rpm += rps
				total_req += rps
				catbox.G_logger.Infof("Requests/sec = %d | Requests/min = %d | Found/min = %d\nTotal Req = %d | Total Found = %d\n", rps, rpm, catbox.G_Found_Per_Min.Load(), total_req, total_found)
				catbox.G_Req_Per_Sec.Store(0)
			case <-min.C:
				total_found += catbox.G_Found_Per_Min.Load()
				rpm = 0
				catbox.G_Found_Per_Min.Store(0)
			}
		}
	}()

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start workers
	for i := 0; i < catbox.G_config.Workers; i++ {
		wg.Add(1) // Correctly add to the wait group here
		go catbox.Worker(ctx, db, idChan, &wg)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	cmdChan := make(chan string)
	go processCommands(cmdChan)

	for {
		select {
		case cmd := <-cmdChan:
			switch cmd {
			case "pause":
				if catbox.G_state == catbox.StateRunning {
					catbox.G_state = catbox.StatePaused
					clearChannel(idChan)
					log.Println("Paused.")
				}
			case "resume":
				if catbox.G_state == catbox.StatePaused {
					catbox.G_state = catbox.StateRunning
					log.Println("Resumed.")
				}
			case "stop":
				catbox.G_state = catbox.StateStopped
				log.Println("Stopping...")
				cancel()
				clearChannel(idChan)
				close(idChan)
				wg.Wait()
				os.Exit(0)
			}
		case <-signalChan:
			catbox.G_state = catbox.StateStopped
			log.Println("Received shutdown signal. Stopping...")
			cancel() // Cancel context to signal workers to stop
			clearChannel(idChan)
			close(idChan)
			wg.Wait()
			os.Exit(0)
		default:
			if catbox.G_state == catbox.StateRunning {
				id := catbox.GenerateID()
				idChan <- id
			} else if catbox.G_state == catbox.StatePaused {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func processCommands(cmdChan chan<- string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading command:", err)
			continue
		}
		cmdChan <- cmd[:len(cmd)-1]
	}
}

func clearChannel(ch chan string) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
