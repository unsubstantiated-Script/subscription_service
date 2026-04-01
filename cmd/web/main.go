package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"subscription_service/data"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = "8080"

func main() {
	//connect to DB
	db := initDB()

	// create sessions
	session := initSession()

	//create logs
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//create channels

	// create waitgroup
	wg := sync.WaitGroup{}

	// set up the app config
	app := Config{
		Session:  session,
		DB:       db,
		InfoLog:  infoLog,
		ErrorLog: errorLog,
		Wait:     &wg,
		Models:   data.New(db),
	}

	//listen for signals
	go app.listenForShutdown()

	// setup mail

	// listen for web connections
	app.serve()
}

func (app *Config) serve() {
	port := os.Getenv("PORT")
	if port == "" {
		port = webPort
	}

	// start http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: app.routes(),
	}

	app.InfoLog.Println("Starting server on port", port)

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}

}

func initDB() *sql.DB {
	conn := connectToDB()
	if conn == nil {
		log.Panic("Could not connect to DB")
	}
	return conn
}

func connectToDB() *sql.DB {
	counts := 0

	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)
		if err == nil {
			log.Println("Postgres connected")
			return connection
		}

		log.Println("Postgres not yet ready...")

		if counts > 10 {
			log.Panic("Could not connect to DB")
			return nil
		}

		log.Println("Backing off for 1 second...")
		time.Sleep(1 * time.Second)
		counts++
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Connected to DB")
	return db, nil
}

func initSession() *scs.SessionManager {
	gob.Register(data.User{})
	//set up sessions
	session := scs.New()
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = sessionCookieSecureFromEnv()
	return session
}

// sessionCookieSecureFromEnv keeps local HTTP development working by default,
// while allowing secure cookies in production via env configuration.
func sessionCookieSecureFromEnv() bool {
	secureOverride := strings.ToLower(strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")))
	switch secureOverride {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return appEnv == "production"
}

func initRedis() *redis.Pool {
	redisPool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
	return redisPool
}

func (app *Config) listenForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	app.shutdown()
	os.Exit(0)
}

func (app *Config) shutdown() {
	// perform any cleanup tasks here
	app.InfoLog.Println("would run cleanup tasks here")

	// block until waitgroup is empty
	app.Wait.Wait()

	app.InfoLog.Println("Closing channels and shutting down")
}
