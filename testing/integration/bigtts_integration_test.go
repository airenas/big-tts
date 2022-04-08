//go:build integration
// +build integration

package doorman

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type config struct {
	uploadURL  string
	statusURL  string
	resultURL  string
	cleanURL  string
	httpClient *http.Client
}

var cfg config

func TestMain(m *testing.M) {
	cfg.uploadURL = getOrFail("UPLOAD_URL")
	cfg.statusURL = getOrFail("STATUS_URL")
	cfg.resultURL = getOrFail("RESULT_URL")
	cfg.cleanURL = getOrFail("CLEAN_URL")

	cfg.httpClient = &http.Client{Timeout: time.Second * 5}

	tCtx, cf := context.WithTimeout(context.Background(), time.Second*20)
	defer cf()
	waitForOpenOrFail(tCtx, cfg.uploadURL)
	waitForOpenOrFail(tCtx, cfg.statusURL)
	waitForOpenOrFail(tCtx, cfg.resultURL)
	waitForOpenOrFail(tCtx, cfg.cleanURL)

	os.Exit(m.Run())
}

func getOrFail(s string) string {
	res := os.Getenv(s)
	if res == "" {
		log.Fatalf("FAIL: no %s set", s)
	}
	return res
}

func waitForOpenOrFail(ctx context.Context, urlWait string) {
	u, err := url.Parse(urlWait)
	if err != nil {
		log.Fatalf("FAIL: can't parse %s", urlWait)
	}
	for {
		if err := listen(net.JoinHostPort(u.Hostname(), u.Port())); err != nil {
			log.Printf("waiting for %s ...", urlWait)
		} else {
			return
		}
		select {
		case <-ctx.Done():
			log.Fatalf("FAIL: can't access %s", urlWait)
			break
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func listen(urlStr string) error {
	log.Printf("dial %s", urlStr)
	conn, err := net.DialTimeout("tcp", urlStr, time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return err
}

func TestUploadLive(t *testing.T) {
	t.Parallel()
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.uploadURL, "/live", nil)), http.StatusOK)
}

func TestUpload(t *testing.T) {
	t.Parallel()
	req := newUploadRequest(t, "file", "t.txt", "olia olia", [][2]string{{"voice", "astra"}})
	checkCode(t, invoke(t, req), http.StatusOK)
}

func TestUpload_FailVoice(t *testing.T) {
	t.Parallel()
	req := newUploadRequest(t, "file", "t.txt", "olia olia", [][2]string{{"voice", "xxx"}})
	checkCode(t, invoke(t, req), http.StatusBadRequest)
}

func TestUpload_FailFile(t *testing.T) {
	t.Parallel()
	req := newUploadRequest(t, "file", "t.doc", "olia olia", [][2]string{{"voice", "astra"}})
	checkCode(t, invoke(t, req), http.StatusBadRequest)
}

func TestStatusLive(t *testing.T) {
	t.Parallel()
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.statusURL, "/live", nil)), http.StatusOK)
}

func TestStatus_Fail(t *testing.T) {
	t.Parallel()
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.statusURL, "/status/olia", nil)), http.StatusBadRequest)
}

func TestResultLive(t *testing.T) {
	t.Parallel()
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.resultURL, "/live", nil)), http.StatusOK)
}

func TestCleanLive(t *testing.T) {
	t.Parallel()
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.cleanURL, "/live", nil)), http.StatusOK)
}

func TestClean_OK(t *testing.T) {
	t.Parallel() 
	resp := invoke(t, newUploadRequest(t, "file", "t.txt", "olia olia", [][2]string{{"voice", "astra"}}))
	checkCode(t, resp, http.StatusOK)
	upResp := result{}
	decode(t, resp, &upResp)
	require.NotEmpty(t, upResp.ID)
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.statusURL, "/status/" + upResp.ID, nil)), http.StatusOK)
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.cleanURL, "/clean/" + upResp.ID, nil)), http.StatusOK)
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.statusURL, "/status/" + upResp.ID, nil)), http.StatusBadRequest)
}

type result struct {
	ID string `json:"id"`
}
func TestStatus_OK(t *testing.T) {
	t.Parallel()
	req := newUploadRequest(t, "file", "t.txt", "olia olia", [][2]string{{"voice", "astra"}})
	resp := invoke(t, req)
	checkCode(t, resp, http.StatusOK)
	upResp := result{}
	decode(t, resp, &upResp)
	require.NotEmpty(t, upResp.ID)
	checkCode(t, invoke(t, newRequest(t, http.MethodGet, cfg.statusURL, "/status/" + upResp.ID, nil)), http.StatusOK)
}

func newRequest(t *testing.T, method string, srv, urlSuffix string, body interface{}) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, srv+urlSuffix, toReader(body))
	require.Nil(t, err, "not nil error = %v", err)
	if body != nil {
		req.Header.Add(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	return req
}

func newUploadRequest(t *testing.T, filep, file, bodyText string, params [][2]string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if file != "" {
		part, _ := writer.CreateFormFile(filep, file)
		_, _ = io.Copy(part, strings.NewReader(bodyText))
	}
	for _, p := range params {
		writer.WriteField(p[0], p[1])
	}
	writer.Close()
	req, err := http.NewRequest(http.MethodPost, cfg.uploadURL+"/upload", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-doorman-requestid", "m:testRequestID")
	return req
}

func invoke(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	resp, err := cfg.httpClient.Do(r)
	require.Nil(t, err, "not nil error = %v", err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func checkCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		b, _ := ioutil.ReadAll(resp.Body)
		require.Equal(t, expected, resp.StatusCode, string(b))
	}
}

func decode(t *testing.T, resp *http.Response, to interface{}) {
	t.Helper()
	require.Nil(t, json.NewDecoder(resp.Body).Decode(to))
}

func toReader(data interface{}) io.Reader {
	bytes, _ := json.Marshal(data)
	return strings.NewReader(string(bytes))
}
