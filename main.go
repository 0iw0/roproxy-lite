package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var timeout, _ = strconv.Atoi(os.Getenv("TIMEOUT"))
var retries, _ = strconv.Atoi(os.Getenv("RETRIES"))
var port = os.Getenv("PORT")

// Webshare Proxy Credentials
var webshareUser = os.Getenv("WEBSHARE_USER")
var websharePass = os.Getenv("WEBSHARE_PASS")

var client *fasthttp.Client

func main() {
	r := router.New()
	r.GET("/{path:*}", proxyHandler)
	r.POST("/{path:*}", proxyHandler)

	// Set up the client with proxy
	client = &fasthttp.Client{
		Dial:          fasthttpproxy.FasthttpHTTPDialer("mpkvgwjp-rotate:65sm7wqm7kr3@p.webshare.io:80"),
		DialDualStack: true,
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
	}

	// Start the server
	port := "8080"
	log.Printf("Starting roproxy-lite on port %s", port)
	if err := fasthttp.ListenAndServe(":"+port, r.Handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func proxyHandler(ctx *fasthttp.RequestCtx) {
	path := ctx.UserValue("path").(string)
	targetURL := fmt.Sprintf("https://www.roblox.com/%s", path)

	req := &fasthttp.Request{}
	req.SetRequestURI(targetURL)
	req.Header.SetMethod(string(ctx.Method()))

	// Copy headers from the original request
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		// Skip certain headers that shouldn't be forwarded
		keyStr := strings.ToLower(string(key))
		if keyStr != "host" && keyStr != "connection" {
			req.Header.Set(string(key), string(value))
		}
	})

	resp := &fasthttp.Response{}
	err := client.Do(req, resp)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadGateway)
		return
	}

	// Copy response headers
	resp.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.Set(string(key), string(value))
	})

	ctx.SetStatusCode(resp.StatusCode())
	ctx.Write(resp.Body())
}
