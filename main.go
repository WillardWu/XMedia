package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"XMedia/internal/core"
	"XMedia/internal/utils"

	"github.com/common-nighthawk/go-figure"
	"github.com/kardianos/service"
)

var productName string
var serviceName string

type program struct {
	core *core.Core
}

func (p *program) Start(s service.Service) (err error) {
	params := map[string]interface{}{
		"product": productName,
	}
	core, err := core.NewCore(params)
	if err != nil || core == nil {
		fmt.Printf("core.NewCore error:%v", err)
		return err
	}
	p.core = core
	p.core.Start()
	return nil
}

func (p *program) Stop(s service.Service) (err error) {
	p.core.Close()
	return nil
}

func main() {
	productName = utils.EXEName()
	serviceName = fmt.Sprintf("%s_Service", productName)

	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceName,
	}
	p := &program{
		core: nil,
	}
	s, err := service.New(p, svcConfig)
	if err != nil {
		fmt.Printf("service.New error:%v", err)
		utils.PauseExit()
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install", "stop":
			if strings.EqualFold(utils.EXEName(), productName) || runtime.GOOS == "windows" {
				figure.NewFigure(productName, "", false).Print()
			}
			fallthrough
		default:
			fmt.Println(svcConfig.Name, os.Args[1], "...")
			if err = service.Control(s, os.Args[1]); err != nil {
				fmt.Printf("error:%v", err)
				utils.PauseExit()
			}
			fmt.Println(svcConfig.Name, os.Args[1], "ok")
		}
		return
	}
	if err = s.Run(); err != nil {
		fmt.Printf("s.Run error:%v", err)
		utils.PauseExit()
	}
}
