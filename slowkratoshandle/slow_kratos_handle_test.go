package slowkratoshandle

import (
	"testing"

	"github.com/orzkratos/authkratos"
)

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)
	m.Run()
}
