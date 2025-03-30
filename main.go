package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var timeout, _ = strconv.Atoi(os.Getenv("TIMEOUT"))
var retries, _ = strconv.Atoi(os.Getenv("RETRIES"))
var port = os.Getenv("PORT")

// Webshare Proxy Credentials
var webshareUser = os.Getenv("WEBSHARE_USER")
var websharePass = os.Getenv("WEBSHARE_PASS")
var proxyURL = "http://" + webshareUser + ":" + websharePass + "@p.webshare.io:80"

func main() {
	h := requestHandler

	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	val, ok := os.LookupEnv("KEY")

	if ok && string(ctx.Request.Header.Peek("PROXYKEY")) != val {
		ctx.SetStatusCode(407)
		ctx.SetBody([]byte("Missing or invalid PROXYKEY header."))
		return
	}

	if len(strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)) < 2 {
		ctx.SetStatusCode(400)
		ctx.SetBody([]byte("URL format invalid."))
		return
	}

	response := makeRequest(ctx, 1)

	defer fasthttp.ReleaseResponse(response)

	body := response.Body()
	ctx.SetBody(body)
	ctx.SetStatusCode(response.StatusCode())
	response.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.Set(string(key), string(value))
	})
}

func makeRequest(ctx *fasthttp.RequestCtx, attempt int) *fasthttp.Response {
	if attempt > retries {
		resp := fasthttp.AcquireResponse()
		resp.SetBody([]byte("Proxy failed to connect. Please try again."))
		resp.SetStatusCode(500)
		return resp
	}

	// Create a new client with proper proxy configuration
	client := &fasthttp.Client{
		ReadTimeout:         time.Duration(timeout) * time.Second,
		MaxIdleConnDuration: 60 * time.Second,
		Dial:                fasthttpproxy.FasthttpHTTPDialerTimeout(proxyURL, time.Duration(timeout)*time.Second),
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// Test URL to show the proxy IP
	req.SetRequestURI("https://api64.ipify.org?format=json")
	req.Header.SetMethod("GET")

	// Remove all headers that might interfere
	req.Header.Del("Connection")
	req.Header.Del("Proxy-Connection")
	req.Header.Set("User-Agent", "RoProxy")

	resp := fasthttp.AcquireResponse()

	err := client.Do(req, resp)
	if err != nil {
		log.Printf("Proxy error (attempt %d): %v", attempt, err)
		fasthttp.ReleaseResponse(resp)
		return makeRequest(ctx, attempt+1)
	}

	return resp
}
