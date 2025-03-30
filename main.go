package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var timeout, _ = strconv.Atoi(os.Getenv("TIMEOUT"))
var retries, _ = strconv.Atoi(os.Getenv("RETRIES"))
var port = os.Getenv("PORT")

// Webshare Proxy Credentials
var webshareUser = os.Getenv("WEBSHARE_USER")
var websharePass = os.Getenv("WEBSHARE_PASS")

var client *fasthttp.Client

func main() {
	h := requestHandler

	client = &fasthttp.Client{
		ReadTimeout:         time.Duration(timeout) * time.Second,
		MaxIdleConnDuration: 60 * time.Second,
		Dial:                fasthttpDialProxy("http://" + webshareUser + ":" + websharePass + "@p.webshare.io:80"),
	}

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

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	url := strings.SplitN(string(ctx.Request.Header.RequestURI())[1:], "/", 2)
	req.SetRequestURI("https://" + url[0] + ".roblox.com/" + url[1])

	req.SetBody(ctx.Request.Body())
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})

	req.Header.Set("User-Agent", "RoProxy")
	req.Header.Del("Roblox-Id")

	// Log request details
	log.Printf("Making request to %s", req.URI())

	resp := fasthttp.AcquireResponse()
	err := client.Do(req, resp)

	// Log the response or error
	if err != nil {
		log.Printf("Error on attempt %d: %v", attempt, err)
		fasthttp.ReleaseResponse(resp)
		return makeRequest(ctx, attempt+1)
	} else {
		log.Printf("Request to %s succeeded with status %d", req.URI(), resp.StatusCode())
		return resp
	}
}

// Helper function to set up proxy dialing
func fasthttpDialProxy(proxyAddr string) fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		return fasthttp.Dial(proxyAddr)
	}
}
