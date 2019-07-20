package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

const (
	GitHubHost  string = "github.com"
	OAuthAppURL string = "https://hub.github.com/"
)

const apiPayloadVersion = "application/vnd.github.v3+json;charset=utf-8"

var UserAgent = "huc"

type Host struct {
	Host        string `toml:"host"`
	User        string `toml:"user"`
	AccessToken string `toml:"access_token"`
	Protocol    string `toml:"protocol"`
	UnixSocket  string `toml:"unix_socket,omitempty"`
}

type verboseTransport struct {
	Transport   *http.Transport
	Verbose     bool
	OverrideURL *url.URL
	Out         io.Writer
	Colorized   bool
}

func (t *verboseTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if t.OverrideURL != nil {
		port := "80"
		if s := strings.Split(req.URL.Host, ":"); len(s) > 1 {
			port = s[1]
		}

		req = cloneRequest(req)
		req.Header.Set("X-Original-Scheme", req.URL.Scheme)
		req.Header.Set("X-Original-Port", port)
		req.Host = req.URL.Host
		req.URL.Scheme = t.OverrideURL.Scheme
		req.URL.Host = t.OverrideURL.Host
	}

	resp, err = t.Transport.RoundTrip(req)

	return
}

func cloneRequest(req *http.Request) *http.Request {
	dup := new(http.Request)
	*dup = *req
	dup.URL, _ = url.Parse(req.URL.String())
	dup.Header = make(http.Header)
	for k, s := range req.Header {
		dup.Header[k] = s
	}
	return dup
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}

func newHttpClient(testHost string, verbose bool, unixSocket string) *http.Client {
	var testURL *url.URL
	if testHost != "" {
		testURL, _ = url.Parse(testHost)
	}
	var httpTransport *http.Transport
	if unixSocket != "" {
		dialFunc := func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		dialContext := func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", unixSocket)
		}
		httpTransport = &http.Transport{
			DialContext:           dialContext,
			DialTLS:               dialFunc,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
		}
	} else {
		httpTransport = &http.Transport{
			Proxy: proxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}
	tr := &verboseTransport{
		Transport:   httpTransport,
		Verbose:     verbose,
		OverrideURL: testURL,
		Out:         os.Stderr,
		Colorized:   isTerminal(os.Stderr),
	}

	return &http.Client{
		Transport: tr,
	}
}

// An implementation of http.ProxyFromEnvironment that isn't broken
func proxyFromEnvironment(req *http.Request) (*url.URL, error) {
	proxy := os.Getenv("http_proxy")
	if proxy == "" {
		proxy = os.Getenv("HTTP_PROXY")
	}
	if proxy == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil || !strings.HasPrefix(proxyURL.Scheme, "http") {
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}

	if err != nil {
		return nil, fmt.Errorf("invalid proxy address %q: %v", proxy, err)
	}

	return proxyURL, nil
}

func normalizeHost(host string) string {
	githubHost := "github.com"
	if host == "" {
		return "github.com"
	} else if strings.EqualFold(host, githubHost) {
		return "api.github.com"
	} else if strings.EqualFold(host, "github.localhost") {
		return "api.github.localhost"
	} else {
		return strings.ToLower(host)
	}
}

type simpleClient struct {
	httpClient     *http.Client
	rootUrl        *url.URL
	PrepareRequest func(*http.Request)
	CacheTTL       int
}

type simpleResponse struct {
	*http.Response
}

type errorInfo struct {
	Message  string       `json:"message"`
	Errors   []fieldError `json:"errors"`
	Response *http.Response
}

func (e *errorInfo) Error() string {
	return e.Message
}

type errorInfoSimple struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

type fieldError struct {
	Resource string `json:"resource"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Field    string `json:"field"`
}

func (res *simpleResponse) Unmarshal(dest interface{}) (err error) {
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	return json.Unmarshal(body, dest)
}

func (res *simpleResponse) ErrorInfo() (msg *errorInfo, err error) {
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	msg = &errorInfo{}
	err = json.Unmarshal(body, msg)
	if err != nil {
		msgSimple := &errorInfoSimple{}
		if err = json.Unmarshal(body, msgSimple); err == nil {
			msg.Message = msgSimple.Message
			for _, errMsg := range msgSimple.Errors {
				msg.Errors = append(msg.Errors, fieldError{
					Code:    "custom",
					Message: errMsg,
				})
			}
		}
	}
	if err == nil {
		msg.Response = res.Response
	}

	return
}

type Client struct {
	Host         *Host
	cachedClient *simpleClient
}

type AuthorizationEntry struct {
	Token string `json:"token"`
}

func (client *Client) apiClient() *simpleClient {
	unixSocket := os.ExpandEnv(client.Host.UnixSocket)
	httpClient := newHttpClient(os.Getenv("HUB_TEST_HOST"), os.Getenv("HUB_VERBOSE") != "", unixSocket)
	apiRoot := client.absolute(normalizeHost(client.Host.Host))
	if !strings.HasPrefix(apiRoot.Host, "api.github.") {
		apiRoot.Path = "/api/v3/"
	}

	return &simpleClient{
		httpClient: httpClient,
		rootUrl:    apiRoot,
	}
}

func (client *Client) absolute(host string) *url.URL {
	u, err := url.Parse("https://" + host + "/")
	if err != nil {
		panic(err)
	} else if client.Host != nil && client.Host.Protocol != "" {
		u.Scheme = client.Host.Protocol
	}
	return u
}

func (client *Client) FindOrCreateToken(user, password, twoFactorCode string) (token string, err error) {
	api := client.apiClient()

	if len(password) >= 40 && isToken(api, password) {
		return password, nil
	}

	params := map[string]interface{}{
		"scopes":   []string{"repo"},
		"note_url": OAuthAppURL,
	}

	api.PrepareRequest = func(req *http.Request) {
		req.SetBasicAuth(user, password)
		if twoFactorCode != "" {
			req.Header.Set("X-GitHub-OTP", twoFactorCode)
		}
	}

	count := 1
	maxTries := 9
	for {
		params["note"], err = authTokenNote(count)
		if err != nil {
			return
		}

		res, postErr := api.PostJSON("authorizations", params)
		if postErr != nil {
			err = postErr
			break
		}

		if res.StatusCode == 201 {
			auth := &AuthorizationEntry{}
			if err = res.Unmarshal(auth); err != nil {
				return
			}
			token = auth.Token
			break
		} else if res.StatusCode == 422 && count < maxTries {
			count++
		} else {
			errInfo, e := res.ErrorInfo()
			if e == nil {
				err = errInfo
			} else {
				err = e
			}
			return
		}
	}

	return
}

func isToken(api *simpleClient, password string) bool {
	api.PrepareRequest = func(req *http.Request) {
		req.Header.Set("Authorization", "token "+password)
	}

	res, _ := api.Get("user")
	if res != nil && res.StatusCode == 200 {
		return true
	}
	return false
}

func authTokenNote(num int) (string, error) {
	n := os.Getenv("USER")

	if n == "" {
		n = os.Getenv("USERNAME")
	}

	if n == "" {
		whoami := exec.Command("whoami")
		whoamiOut, err := whoami.Output()
		if err != nil {
			return "", err
		}
		n = strings.TrimSpace(string(whoamiOut))
	}

	h, err := os.Hostname()
	if err != nil {
		return "", err
	}

	if num > 1 {
		return fmt.Sprintf("hub for %s@%s %d", n, h, num), nil
	}

	return fmt.Sprintf("hub for %s@%s", n, h), nil
}

func (c *simpleClient) performRequest(method, path string, body io.Reader, configure func(*http.Request)) (*simpleResponse, error) {
	url, err := url.Parse(path)
	if err == nil {
		url = c.rootUrl.ResolveReference(url)
		return c.performRequestUrl(method, url, body, configure)
	} else {
		return nil, err
	}
}

func (c *simpleClient) performRequestUrl(method string, url *url.URL, body io.Reader, configure func(*http.Request)) (res *simpleResponse, err error) {
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return
	}
	if c.PrepareRequest != nil {
		c.PrepareRequest(req)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", apiPayloadVersion)

	if configure != nil {
		configure(req)
	}

	httpResponse, err := c.httpClient.Do(req)
	if err != nil {
		return
	}
	res = &simpleResponse{httpResponse}

	return
}

func (c *simpleClient) Get(path string) (*simpleResponse, error) {
	return c.performRequest("GET", path, nil, nil)
}

func (c *simpleClient) PostJSON(path string, payload interface{}) (*simpleResponse, error) {
	return c.jsonRequest("POST", path, payload, nil)
}

func (c *simpleClient) jsonRequest(method, path string, body interface{}, configure func(*http.Request)) (*simpleResponse, error) {
	json, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(json)

	return c.performRequest(method, path, buf, func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if configure != nil {
			configure(req)
		}
	})
}
