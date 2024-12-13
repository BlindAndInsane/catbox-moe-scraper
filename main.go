package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/webhook"
	"github.com/sirupsen/logrus"

	"catbox-scraper/catbox"
)

func main() {
	if err := catbox.LoadConfig(); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	db, err := catbox.InitDB(catbox.G_config.Database)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	if catbox.G_config.UseProxies {
		if err := initializeProxyManager(); err != nil {
			log.Fatalf("Error initializing proxy manager: %v", err)
		}
	}

	if catbox.G_config.WebhookEnabled {
		if err := initializeWebhookClient(); err != nil {
			log.Fatalf("Error initializing webhook client: %v", err)
		}
	}

	catbox.EnsureDownloadPathExists(catbox.G_config.DownloadPath)

	var wg sync.WaitGroup
	idChan := make(chan string, catbox.G_config.Workers)

	startMonitoring()
	startWorkers(db, idChan, &wg)
	handleCommandsAndSignals(idChan, &wg)
}

func initializeProxyManager() error {
	var err error
	catbox.G_proxyManager, err = catbox.InitProxyManager(catbox.G_config.ProxyFile)
	return err
}

func initializeWebhookClient() error {
	webhookLogger := logrus.New()
	webhookLogger.SetLevel(logrus.ErrorLevel)

	client, err := webhook.NewWithURL(
		catbox.G_config.WebhookURL,
		webhook.WithLogger(webhookLogger),
		webhook.WithRestClientConfigOpts(
			rest.WithRateRateLimiterConfigOpts(
				rest.WithRateLimiterLogger(webhookLogger),
			),
		),
	)
	if err != nil {
		return err
	}

	catbox.G_webhook_client = client
	return nil
}

func startMonitoring() {
	go func() {
		secTicker := time.NewTicker(time.Second)
		minTicker := time.NewTicker(time.Minute)
		defer secTicker.Stop()
		defer minTicker.Stop()

		var rpm, totalReq, totalFound int64
		for {
			select {
			case <-secTicker.C:
				rps := catbox.G_Req_Per_Sec.Load()
				rpm += rps
				totalReq += rps
				fpm := catbox.G_Found_Per_Min.Load()
				catbox.G_logger.Infof("Requests/sec: %d | Requests/min: %d | Found/min: %d | Total Req: %d | Total Found: %d",
					rps, rpm, fpm, totalReq, totalFound+fpm)
				catbox.G_Req_Per_Sec.Store(0)
			case <-minTicker.C:
				totalFound += catbox.G_Found_Per_Min.Load()
				rpm = 0
				catbox.G_Found_Per_Min.Store(0)
			}
		}
	}()
}

func startWorkers(db *sql.DB, idChan chan string, wg *sync.WaitGroup) {
	for i := 0; i < catbox.G_config.Workers; i++ {
		go catbox.Worker(db, idChan, wg)
	}
}

func handleCommandsAndSignals(idChan chan string, wg *sync.WaitGroup) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	cmdChan := make(chan string)
	go processCommands(cmdChan)

	for {
		select {
		case cmd := <-cmdChan:
			switch cmd {
			case "pause":
				pauseProcessing(idChan)
			case "resume":
				resumeProcessing()
			case "stop":
				shutdown(idChan, wg)
			}
		case <-signalChan:
			shutdown(idChan, wg)
		default:
			if catbox.G_state == catbox.StateRunning {
				idChan <- catbox.GenerateID()
			} else if catbox.G_state == catbox.StatePaused {
				time.Sleep(time.Second)
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

func pauseProcessing(idChan chan string) {
	if catbox.G_state == catbox.StateRunning {
		catbox.G_state = catbox.StatePaused
		clearChannel(idChan)
		log.Println("Paused.")
	}
}

func resumeProcessing() {
	if catbox.G_state == catbox.StatePaused {
		catbox.G_state = catbox.StateRunning
		log.Println("Resumed.")
	}
}

func shutdown(idChan chan string, wg *sync.WaitGroup) {
	catbox.G_state = catbox.StateStopped
	log.Println("Shutting down...")
	clearChannel(idChan)
	close(idChan)
	wg.Wait()
	os.Exit(0)
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
