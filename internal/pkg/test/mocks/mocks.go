package mocks

import (
	"testing"

	"github.com/petergtz/pegomock/v4"
)

//go:generate pegomock generate --package=mocks --output=statusProvider.go github.com/airenas/big-tts/internal/pkg/statusservice StatusProvider

//go:generate pegomock generate --package=mocks --output=fileSaver.go github.com/airenas/big-tts/internal/pkg/upload FileSaver

//go:generate pegomock generate --package=mocks --output=msgSender.go github.com/airenas/big-tts/internal/pkg/upload MsgSender

//go:generate pegomock generate --package=mocks --output=requestSaver.go github.com/airenas/big-tts/internal/pkg/upload RequestSaver

//go:generate pegomock generate --package=mocks --output=fileReader.go github.com/airenas/big-tts/internal/pkg/result FileReader

//go:generate pegomock generate --package=mocks --output=fileNameProvider.go github.com/airenas/big-tts/internal/pkg/result FileNameProvider

//go:generate pegomock generate --package=mocks --output=worker.go github.com/airenas/big-tts/internal/pkg/synthesize Worker

//go:generate pegomock generate --package=mocks --output=statusSaver.go github.com/airenas/big-tts/internal/pkg/synthesize StatusSaver

//go:generate pegomock generate --package=mocks --output=cleaner.go github.com/airenas/big-tts/internal/pkg/clean Cleaner

//go:generate pegomock generate --package=mocks --output=emailSender.go github.com/airenas/big-tts/internal/pkg/inform Sender

//go:generate pegomock generate --package=mocks --output=emailMaker.go github.com/airenas/big-tts/internal/pkg/inform EmailMaker

//go:generate pegomock generate --package=mocks --output=emailRetriever.go github.com/airenas/big-tts/internal/pkg/inform EmailRetriever

//go:generate pegomock generate --package=mocks --output=locker.go github.com/airenas/big-tts/internal/pkg/inform Locker

// AttachMockToTest register pegomock verification to be passed to testing engine
func AttachMockToTest(t *testing.T) {
	pegomock.RegisterMockFailHandler(handleByTest(t))
}

func handleByTest(t *testing.T) pegomock.FailHandler {
	return func(message string, callerSkip ...int) {
		t.Helper()
		if message != "" {
			t.Error(message)
		}
	}
}
