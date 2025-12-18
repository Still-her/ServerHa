package config

import (
	"flag"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var configFile = flag.String("config", "./config/ha.yaml", "配置文件路径")
var Instance *Config

func init() {
	Instance = &Config{}
	if yamlFile, err := ioutil.ReadFile(*configFile); err != nil {
		logrus.Error(err)
	} else if err = yaml.Unmarshal(yamlFile, Instance); err != nil {
		logrus.Error(err)
	}
	return
}

type IPNODE struct {
	Ip      string `yaml:"ip"`      //ip列表
	Netment string `yaml:"netment"` //网元名字
}

type Config struct {
	Logs struct {
		Isshowdb bool   `yaml:"isshowdb"`
		Isonlyfa bool   `yaml:"isonlyfa"`
		Logdfile string `yaml:"logdfile"`
		Loglevel int8   `yaml:"loglevel"`
		Logmfile string `yaml:"logmfile"`
		Maxsize  int    `yaml:"maxsize" json:"maxsize"`
		Maxback  int    `yaml:"maxback" json:"maxback"`
		Maxdays  int    `yaml:"maxdays" json:"maxdays"`
	} `yaml:"logs"`

	Httpvip  string `yaml:"httpvip"`  //对外ip
	Httpport string `yaml:"httpport"` //对外端口

	Seriesne struct { //串联网元
		Detectmult int      `yaml:"detectmult"` //报文最大失效的个数
		Vip        []IPNODE `yaml:"vip"`
	} `yaml:"seriesne"`

	Station struct { //主备站点间配置
		Priority int  `yaml:"priority"` //优先级 1-255
		Preeempt bool `yaml:"preeempt"` //是否抢占模式

		Self struct { //本站点
			Vip string `yaml:"vip"` //本站点vip
			Pip string `yaml:"pip"` //本站点节点物理ip
			Sip string `yaml:"sip"` // #本站点另一个物理ip
		} `yaml:"self"`
		Paralle struct { //对端站点
			Detectmult int    `yaml:"detectmult"` //报文最大失效的个数
			Allupmode  int    `yaml:"allupmode"`  //物理ip全部正常，vip失效
			Oneupmode  int    `yaml:"oneupmode"`  //物理ip一个正常，vip失效
			Zeroupmode int    `yaml:"zeroupmode"` //物理ip全部失效，vip失效
			Vip        string `yaml:"vip"`        //对端站点vip

			Pip []IPNODE `yaml:"pip"` //对端站点物理ip
		} `yaml:"paralle"`
	} `yaml:"station"`

	Internal struct { //站内节点间配置
		Iterpip  string `yaml:"iterpip"`  //虚拟IP
		Routerid int    `yaml:"routerid"` //路由ID
		Priority int    `yaml:"priority"` //路由优先级
		Preeempt bool   `yaml:"preeempt"` //是否抢占模式
		Netdev   string `yaml:"netdev"`   //网卡设备
		Itervip  string `yaml:"itervip"`  //虚拟IP
		Maskbit  string `yaml:"maskbit"`  //掩码位数
		Interval int    `yaml:"interval"` //时间间隔，单位ms
		Testtime int    `yaml:"testtime"` //时间间隔，单位ms
	} `yaml:"internal"`
}
