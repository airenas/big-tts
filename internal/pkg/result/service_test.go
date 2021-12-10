package result

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/airenas/big-tts/internal/pkg/test/mocks"
	"github.com/labstack/echo/v4"
	"github.com/petergtz/pegomock"
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
		{name: "OK", args: args{data: &Data{Reader: mocks.NewMockFileReader(),
			NameProvider: mocks.NewMockFileNameProvider()}}, wantErr: false},
		{name: "Fail reader", args: args{data: &Data{Reader: nil,
			NameProvider: mocks.NewMockFileNameProvider()}}, wantErr: true},
		{name: "Fail provider", args: args{data: &Data{Reader: mocks.NewMockFileReader(),
			NameProvider: nil}}, wantErr: true},
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
	readerMock    *mocks.MockFileReader
	nProviderMock *mocks.MockFileNameProvider
	tData         *Data
	tEcho         *echo.Echo
	tResp         *httptest.ResponseRecorder
)

func initTest(t *testing.T) {
	mocks.AttachMockToTest(t)
	readerMock = mocks.NewMockFileReader()
	nProviderMock = mocks.NewMockFileNameProvider()
	tData = &Data{}
	tData.Reader = readerMock
	tData.NameProvider = nProviderMock
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
	req := httptest.NewRequest(http.MethodPost, "/result/1", nil)
	testCode(t, req, 405)
}

func Test_Returns(t *testing.T) {
	initTest(t)

	tf, err := ioutil.TempFile("", "tmpFile.txt")
	tf.WriteString("olia")
	assert.Nil(t, err)
	defer os.RemoveAll(tf.Name())

	pegomock.When(readerMock.Load(pegomock.AnyString())).ThenReturn(tf, nil)
	req := httptest.NewRequest(http.MethodGet, "/result/1", nil)
	resp := testCode(t, req, 200)
	bytes, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "olia", string(bytes))
}

func Test_404(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/result/", nil)
	testCode(t, req, 404)
}

func Test_Fails_NameProvider(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/result/1", nil)
	pegomock.When(nProviderMock.GetResultFile(pegomock.AnyString())).ThenReturn("", errors.New("err"))
	testCode(t, req, http.StatusBadRequest)
}

func Test_Fails_Reader(t *testing.T) {
	initTest(t)
	req := httptest.NewRequest(http.MethodGet, "/result/1", nil)
	pegomock.When(readerMock.Load(pegomock.AnyString())).ThenReturn(nil, errors.New("err"))

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
