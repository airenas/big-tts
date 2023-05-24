package statusservice

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airenas/big-tts/internal/pkg/persistence"
	"github.com/airenas/big-tts/internal/pkg/test/mocks"
	"github.com/labstack/echo/v4"
	"github.com/petergtz/pegomock/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	providerMock *mocks.MockStatusProvider
	tData        *Data
	tEcho        *echo.Echo
	tResp        *httptest.ResponseRecorder
)

func TestStartWebServer_Fail(t *testing.T) {
	err := StartWebServer(&Data{})
	assert.NotNil(t, err)
}

func initTest(t *testing.T) {
	mocks.AttachMockToTest(t)
	providerMock = mocks.NewMockStatusProvider()
	tData = &Data{}
	tData.StatusProvider = providerMock
	tEcho = initRoutes(tData)
	tResp = httptest.NewRecorder()
}

func TestWrongPath(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest("GET", "/invalid", nil)
	testCode(t, req, 404)
}

func TestWrongMethod(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodPost, "/status/12", nil)
	testCode(t, req, 405)
}

func Test_Returns(t *testing.T) {
	initTest(t)
	pegomock.When(providerMock.Get(pegomock.Any[string]())).ThenReturn(&persistence.Status{ID: "10", Status: "olia"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/status/10", nil)
	resp := testCode(t, req, 200)
	bytes, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(bytes), `"status":"olia"`)
	mID := providerMock.VerifyWasCalled(pegomock.Once()).Get(pegomock.Any[string]()).GetCapturedArguments()
	assert.Equal(t, "10", mID)
}

func Test_Returns_Error(t *testing.T) {
	initTest(t)
	pegomock.When(providerMock.Get(pegomock.Any[string]())).ThenReturn(&persistence.Status{ID: "10", Status: "olia", Error: "err"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/status/10", nil)
	resp := testCode(t, req, 200)
	bytes, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(bytes), `"error":"err"`)
	mID := providerMock.VerifyWasCalled(pegomock.Once()).Get(pegomock.Any[string]()).GetCapturedArguments()
	assert.Equal(t, "10", mID)
}

func Test_Fails(t *testing.T) {
	initTest(t)
	pegomock.When(providerMock.Get(pegomock.Any[string]())).ThenReturn(nil, errors.New("olia"))
	req := httptest.NewRequest(http.MethodGet, "/status/10", nil)
	testCode(t, req, 500)
}

func Test_Fails_NoStatus(t *testing.T) {
	initTest(t)
	pegomock.When(providerMock.Get(pegomock.Any[string]())).ThenReturn(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/status/10", nil)
	testCode(t, req, 400)
}

func Test_Live(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	testCode(t, req, 200)
}

func testCode(t *testing.T, req *http.Request, code int) *httptest.ResponseRecorder {
	tEcho.ServeHTTP(tResp, req)
	assert.Equal(t, code, tResp.Code)
	return tResp
}
