package deepjoy

//go:generate go-mockgen github.com/efritz/deepjoy -o mock_test.go -i Conn -i Pool

import (
	"testing"

	"github.com/aphistic/sweet"
	"github.com/aphistic/sweet-junit"
	. "github.com/onsi/gomega"
)

var testLogger = NewNilLogger()

func TestMain(m *testing.M) {
	RegisterFailHandler(sweet.GomegaFail)

	sweet.Run(m, func(s *sweet.S) {
		s.RegisterPlugin(junit.NewPlugin())

		s.AddSuite(&PoolSuite{})
		s.AddSuite(&ClientSuite{})
	})
}
