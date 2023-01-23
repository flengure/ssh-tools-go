package tools

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
)

type Job map[string]string

type Jobs map[string]map[string]string

type Host struct {
	Desc    string `json:"Desc"`
	Editors Jobs   `json:"Editors"`
	Viewers Jobs   `json:"Viewers"`
}

type Hosts map[string]Host

type Config struct {
	Hosts Hosts  `json:"Hosts"` // map of hosts
	Host  string `json:"Host"`  // last selected host
	File  string // config file path
}

func (c *Config) DefaultHost() string {
	if c.Host != "" {
		if _, ok := c.Hosts[c.Host]; ok {
			return c.Host
		}
	}
	for k := range c.Hosts {
		return k
	}
	return ""
}

func (c *Config) Json() (string, error) {

	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return "", err
	}

	return string(b), nil

}

// func (c *Config) Copy(host string) error {

// 	// c.Hosts[host] = c.Hosts[host

// 	bytes, err := json.MarshalIndent(c, "", "\t")
// 	if err != nil {
// 		return err
// 	}

// 	err = os.WriteFile(file, bytes, 0644)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }

func (c *Config) SaveAs(file string) error {

	// delete(c.Hosts, "")

	bytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(file, bytes, 0644)
	if err != nil {
		return err
	}

	return nil

}

func (c *Config) Save() error {

	err := c.SaveAs(c.File)
	if err != nil {
		return err
	}

	return nil

}

func LoadConfigFrom(file string) (Config, error) {

	config1 := Config{}
	// config2 := Config{}

	_, err := os.Stat(file)
	if err != nil {
		return config1, err
	}

	bytes1, err := os.ReadFile(file)
	if err != nil {
		return config1, err
	}

	json.Unmarshal(bytes1, &config1)

	if reflect.DeepEqual(config1, Config{}) {
		return Config{}, errors.New("bad config: " + file)
	}

	// bytes2, _ := json.Marshal(config1)
	// json.Unmarshal(bytes2, &config2)

	// fmt.Println(reflect.DeepEqual(config1, config2))

	// res := bytes.Compare(bytes1, bytes2)

	// if res == 0 {

	// 	config.File = file
	// 	fmt.Println("good file")

	// 	return config, nil
	// }
	// fmt.Println("bad file: ", file)
	return config1, nil
	// fmt.Println(string(bytes1))

	//fmt.Println(maps.Keys(config.Hosts))
	// fmt.Println(string(bytes))
}

func viewAclCmd(s string) string {
	return strings.Join(strings.Fields(`
		c=nft;
		if command -v $c &> /dev/null; then
			b=$($c list sets);
			p=$(for i in ipv6 ipv4 mac; do
					t=`+s+`_$i; 
					printf "%s" "$b" | grep -q "set $t {" && {
						$c list set inet fw4 $t; 
					};
				done;
			);
			[ "$p" != "" ] && {
				printf "\n%s\n%s\n%s\n" "nfsets" "------" "$p";
			};
		else 			
			printf "%s\n" "$c command not found";
		fi;
		f=/etc/dnsmasq.d/`+s+`.conf;
		if [ -f "$f" ]; then c=$(cat "$f"); fi;`,
	), " ")
}

func cmdArp() string {
	var sb strings.Builder
	sb.WriteString("f='%-18s %-17s\\n'; ")
	sb.WriteString("ip neigh show | ")
	sb.WriteString("awk -v f=\"$f\" 'BEGIN{ ")
	sb.WriteString("printf f, \"-----------------\", \"---------------\";")
	sb.WriteString("printf f, \"Hardware Address\", \"IP Adress\";")
	sb.WriteString("printf f, \"-----------------\", \"---------------\"}")
	sb.WriteString("/REACHABLE/{printf f, $5, $1}'")
	return sb.String()
}

var viewAcl = Jobs{
	"src_accept": {
		"desc": "Show hosts allowed internet access",
		"cmd":  viewAclCmd("src_accept"),
	},
	"src_reject": {
		"desc": "Show hosts denied internet access",
		"cmd":  viewAclCmd("src_reject"),
	},
	"dest_accept": {
		"desc": "Show hosts allowed internet access",
		"cmd":  viewAclCmd("src_reject"),
	},
	"dest_reject": {
		"desc": "Show hosts denied internet access",
		"cmd":  viewAclCmd("src_reject"),
	},
	"arp": {
		"desc": "Show ip neigbors mac addresses",
		"cmd":  cmdArp(),
	},
}

var editAcl = Jobs{
	"src_accept": {
		"desc": "Edit hosts allowed internet access and restart the firewall",
		"file": "/etc/firewall/user/src_accept.txt",
		"cmd":  "fw4 restart",
	},
	"src_reject": {
		"desc": "Edit hosts denied internet access and restart the firewall",
		"file": "/etc/firewall/user/src_reject.txt",
		"cmd":  "fw4 restart",
	},
	"dest_accept": {
		"desc": "Edit hosts allowed internet access and restart the firewall",
		"file": "/etc/firewall/user/dest_accept.txt",
		"cmd":  "fw4 restart",
	},
	"dest_reject": {
		"desc": "Edit hosts denied internet access and restart the firewall",
		"file": "/etc/firewall/user/dest_accept.txt",
		"cmd":  "fw4 restart",
	},
	"hosts": {
		"desc": "Edit the hosts file",
		"file": "/etc/hosts",
	},
	"authorized_keys": {
		"desc": "Edit the authorized_keys file",
		"file": "/etc/dropbear/authorized_keys",
	},
}

func NewConfig() *Config {
	return &Config{
		Host: "",
		File: "config.json",
	}
}

func NewConfigAcl() *Config {
	// returns a default config object for my openwrt acl script
	config := NewConfig()
	config.Hosts = Hosts{
		"": Host{
			Desc:    "OpenWRT",
			Editors: editAcl,
			Viewers: viewAcl,
		},
	}
	return config
}

var editNomic = Jobs{
	"hosts": {
		"desc": "Edit the hosts file",
		"file": "/etc/hosts",
	},
}
var viewNomic = Jobs{
	"status": {
		"desc": "Show status information",
		"cmd":  "./tstatus/tstatus",
	},
	"daily summary": {
		"desc": "Show Hosts denied internet access",
		"cmd":  "/root/autodelegate/daily-summary.sh 14",
	},
}

func NewConfigNomic() *Config {
	// returns a default config object for my nomic
	config := NewConfig()
	config.Hosts = Hosts{
		"": Host{
			Desc:    "Ubuntu - nomic",
			Editors: editNomic,
			Viewers: viewNomic,
		},
	}
	return config
}
