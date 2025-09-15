package conf

import (
	"XMedia/internal/utils"
	"fmt"

	"gopkg.in/ini.v1"
)

const CONFIG_FILE = "xmedia.ini"

// General
type GeneralConf struct {
	ReadTimeoutRaw  string   `ini:"readTimeout"`
	WriteTimeoutRaw string   `ini:"writeTimeout"`
	WriteQueueSize  int      `ini:"writeQueueSize"`
	ReadTimeout     Duration `ini:"-" json:"-"` // filled by Check()
	WriteTimeout    Duration `ini:"-" json:"-"` // filled by Check()
}

// Log
type LogConf struct {
	LogMaxSize   int `ini:"logMaxSize"`
	LogMaxBackup int `ini:"logMaxBackup"`
	LogQueueSize int `ini:"logQueueSize"`
	LogSaveDays  int `ini:"logSaveDays"`
}

// Rtsp
type RtspConf struct {
	Rtsp           bool           `ini:"rtsp"`
	RtspTransports RTSPTransports `ini:"-" json:"-"` // filled by Check()
	RtspAddress    string         `ini:"rtspAddress"`

	RtspTransportsRaw string `ini:"rtspTransports"`
}

type Config struct {
	Ini *ini.File `ini:"-" json:"-"`

	// General
	General GeneralConf `ini:"general"`

	// Log
	Log LogConf `ini:"log"`

	// Rtsp
	Rtsp RtspConf `ini:"rtsp"`
}

func (c *Config) Check() error {

	err := c.General.ReadTimeout.Marshal(c.General.ReadTimeoutRaw)
	if err != nil {
		return err
	}

	err = c.General.WriteTimeout.Marshal(c.General.WriteTimeoutRaw)
	if err != nil {
		return err
	}

	err = c.Rtsp.RtspTransports.UnmarshalEnv("", c.Rtsp.RtspTransportsRaw)
	if err != nil {
		return err
	}

	return nil
}

func Load(file string) (cfg *Config, err error) {
	iFile := utils.FileTotalPath(file)
	if !utils.Exist(iFile) {
		return nil, fmt.Errorf("Config.Load %s error:文件不存在", file)
	}
	cfg = &Config{}
	cfg.Ini, err = ini.Load(iFile)
	if err != nil {
		return nil, fmt.Errorf("Config.Load %s error:%v", file, err)
	}
	cfg.Ini.NameMapper = nil
	err = cfg.Ini.MapTo(cfg)
	if err != nil {
		return nil, fmt.Errorf("Config.Load %s error:%v", file, err)
	}
	err = cfg.Check()
	if err != nil {
		return nil, fmt.Errorf("Config.Load %s error:%v", file, err)
	}

	return cfg, nil
}
