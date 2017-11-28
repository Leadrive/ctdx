package comm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert.True(t, true)
	conf := new(Conf)
	conf.Parse("../configure.toml")
	assert.Equal(t, "DEBUG", conf.App.Logger.Level)
	assert.Equal(t, "datochan", conf.App.Logger.Name)
}
