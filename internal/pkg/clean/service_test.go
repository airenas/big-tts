package clean

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airenas/big-tts/internal/pkg/test/mocks"
	"github.com/labstack/echo/v4"
	"github.com/petergtz/pegomock/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_validate(t *testing.T) {
	type args struct {
		data *Data
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "OK", args: args{data: &Data{Cleaner: mocks.NewMockCleaner()}}, wantErr: false},
		{name: "Fail cleaner", args: args{data: &Data{Cleaner: nil}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("StartWebServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var (
	cleanMock *mocks.MockCleaner
	tData     *Data
	tEcho     *echo.Echo
	tResp     *httptest.ResponseRecorder
)

func initTest(t *testing.T) {
	mocks.AttachMockToTest(t)
	cleanMock = mocks.NewMockCleaner()
	tData = &Data{}
	tData.Cleaner = cleanMock
	tEcho = initRoutes(tData)
	tResp = httptest.NewRecorder()
}

func TestWrongPath(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	testCode(t, req, 404)
}

func TestWrongMethod(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodPost, "/delete/1", nil)
	testCode(t, req, 405)
}

func Test_Delete(t *testing.T) {
	initTest(t)

	pegomock.When(cleanMock.Clean(pegomock.Any[string]())).ThenReturn(nil)
	req := httptest.NewRequest(http.MethodDelete, "/delete/1", nil)
	resp := testCode(t, req, 200)
	bytes, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "deleted", string(bytes))
}

func Test_404(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/delete/", nil)
	testCode(t, req, 404)
}

func Test_Fails_Clean(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodDelete, "/delete/1", nil)
	pegomock.When(cleanMock.Clean(pegomock.Any[string]())).ThenReturn(errors.New("err"))
	testCode(t, req, http.StatusInternalServerError)
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
