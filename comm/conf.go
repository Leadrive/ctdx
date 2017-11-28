package comm

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Conf struct {
	App struct {
		Mode string `toml:"mode"`
		DataPath string `toml:"data_path"`

		Logger struct {
			Level string `toml:"level"`
			Name string `toml:"name"`
		} `toml:"logger"`

		Urls struct {
			StockFin string `toml:"stock_fin"`
			FinListFile string `toml:"fin_list_file"`
		} `toml:"urls"`

		Files struct {
			StockList string `toml:"stock_list"`
			StockBonus string `toml:"stock_bonus"`
			StockDay string `toml:"stock_day"`
			StockMin string `toml:"stock_min"`
			StockReport string `toml:"stock_report"`
		} `toml:"files"`

	} `toml:"app"`

	Tdx struct {
		DataHost string `toml:"data_host"`
		MonitorHost string `toml:"monitor_host"`
	} `toml:"tdx"`
}


///////////////////////////////////////////////////////////////////

// `Global` section

///////////////////////////////////////////////////////////////////

// Neo Application configuration
func (c *Conf) loadDefaults() {

	// app
	c.App.Logger.Level = "INFO"
	c.App.Logger.Name = "ctdx"
	c.App.Mode = "debug"
}

// Will try to parse TOML configuration file.

func (c *Conf) Parse(path string) {
	c.loadDefaults()
	if path == "" {
		log.Printf("Loaded configuration defaults")
		return
	}

	if _, err := toml.DecodeFile(path, c); err != nil {
		panic(err)
	}
}