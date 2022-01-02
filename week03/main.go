package main

import (
	"os"
	"os/signal"
	"syscall"

	"log"
	"time"
	"context"
	"net/http"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func main() {
	eg, ctx 		:= errgroup.WithContext(context.Background())

	mux 			:= http.NewServeMux()
	serverOut 		:= make(chan struct{})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("server test"))
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		serverOut <- struct{}{}
	})

	server 			:= http.Server{
		Handler : 	mux,
		Addr :    	":8080",
	}

	// 开始监听
	eg.Go(func() (err error) {
		log.Printf("[info] start http server listening %s \n", server.Addr)
		return server.ListenAndServe()
	})

	// 捕获到 os 退出信号将会退出
	eg.Go(func() error {
		quit 		:= make(chan os.Signal, 0)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-quit:
			return errors.Errorf("quit_ch: get os signal: %v", sig)
		}
	})

	// http 控制
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			log.Println("ch: errgroup Done...")
		case <-serverOut:
			log.Println("ch: server(shutdown) out...")
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Println("Server shutdown")
		return server.Shutdown(timeoutCtx)
	})

	log.Printf("errgroup exit: %+v\n", eg.Wait())
}

