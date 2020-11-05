package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/coocood/freecache"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)

var cache *freecache.Cache

type ResultStruct struct {
	Card    string `json:"card_id"`
	Version string `json:"version"`
}

type MyResponse struct {
	Status string       `json:"status"`
	Result ResultStruct `json:"results"`
}

type ScanResponse struct {
	Status string `json:"status"`
	Result bool   `json:"results"`
}

func ScanNow() {
	cmd := exec.Command("sh", "/opt/vitinvest/ace.pos/scan.sh")
	cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
}

func HelloServer(w http.ResponseWriter, req *http.Request) {
	aaa, _ := json.Marshal(MyResponse{Status: "OK", Result: ResultStruct{Card: CurrentCard(), Version: "go"}})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(aaa)
}

func ScanServer(w http.ResponseWriter, req *http.Request) {
	scanStarted := false

	_, err := cache.Get([]byte("abc"))
	if err != nil {
		// lock scan for 10 second
		cache.Set([]byte("abc"), []byte("def"), 10)
		if _, err := os.Stat("/opt/vitinvest/ace.pos/scan.sh"); err == nil {
			go ScanNow()
			scanStarted = true
		}
	}

	aaa, _ := json.Marshal(ScanResponse{Status: "OK", Result: scanStarted})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(aaa)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	// register client
	clients[ws] = true
}

func SendCard(card []byte) {
	// RemoteAddr, err := net.ResolveUDPAddr("udp", "192.168.1.55:6000")
	RemoteAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6000")
	conn, err := net.DialUDP("udp", nil, RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	_, err = conn.Write(card)

	if err != nil {
		log.Println(err)
	}
}

func echo(broadcast chan string) {
	for {
		val := <-broadcast

		// SendCard([]byte(val))

		for client := range clients {
			err := client.WriteJSON(MyResponse{Status: "OK", Result: ResultStruct{Card: val, Version: "go"}})
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func checkHosts() {
	dat, _ := ioutil.ReadFile("/etc/hosts")
	if !bytes.Contains(dat, []byte("127.0.0.1 cardreader.vitinvest.com")) {
		dat = append(dat, []byte("\r\n127.0.0.1 cardreader.vitinvest.com\r\n")...)
		ioutil.WriteFile("/etc/hosts", dat, 0644)
	}
}

func main() {
	checkHosts()
	cache = freecache.NewCache(200)

	cert := []byte(`-----BEGIN CERTIFICATE-----
MIIEkTCCA3mgAwIBAgIJAMHEtnIPaLi8MA0GCSqGSIb3DQEBCwUAMIGWMQswCQYD
VQQGEwJSVTEPMA0GA1UECAwGTW9zY293MQ8wDQYDVQQHDAZNb3Njb3cxFDASBgNV
BAoMC1ZpdGFsaW52ZXN0MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRQwEgYDVQQD
DAtWaXRhbGludmVzdDEhMB8GCSqGSIb3DQEJARYSaW5mb0BhY2VzeXN0ZW0uY29t
MB4XDTE3MDQyNTEwMTAzOFoXDTMxMDEwMjEwMTAzOFowgaMxCzAJBgNVBAYTAlJV
MQ8wDQYDVQQIDAZNb3Njb3cxDzANBgNVBAcMBk1vc2NvdzEUMBIGA1UECgwLVml0
YWxpbnZlc3QxFjAUBgNVBAsMDUlUIERlcGFydG1lbnQxITAfBgNVBAMMGGNhcmRy
ZWFkZXIudml0aW52ZXN0LmNvbTEhMB8GCSqGSIb3DQEJARYSaW5mb0BhY2VzeXN0
ZW0uY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzj340V7yc0TS
OA2MvCcvE1aHcF/jPPj/ptdDGU62ADf6++mpCv05ZrmTwhnEgMMBGikj8h9Uoutk
286Qcjbd6V6NECPAfx1GbR4C2NZYWrdFxx5J3L6Qb+bLcXxZ4SZ1HsLCJZQ3ekfy
ORJbUa491E0+vIyHL0VceplZxe7pWxC74Z3nUqUqXBnYkjMnrweeeyOPRa/pESs1
59jn5q6KQq7kV1mqtTKOfUitcIQT36t2QHP3zALMImZN8/pW4dU1OXXU6O4FfaEo
rNTIijOFwq3gfuKiBrQ1Lpy0OIHvAyQhI2wf6VowkHMTbftf4+hn8LkPX9tupPVi
8xE3+qeVbQIDAQABo4HSMIHPMAkGA1UdEwQCMAAwCwYDVR0PBAQDAgXgMIG0BgNV
HREEgawwgamCCWxvY2FsaG9zdIIRc3J2LmFjZXN5c3RlbS5jb22CFGNhc2luby5h
Y2VzeXN0ZW0uY29tghNzcnY3OC5hY2VzeXN0ZW0uY29tghNzcnYzNS5hY2VzeXN0
ZW0uY29tghF3d3cuYWNlc3lzdGVtLmNvbYIYY2FyZHJlYWRlci52aXRpbnZlc3Qu
Y29thwTAqAFOhwTAqAEjhwTAqAEYhwR/AAABhwSsEAABMA0GCSqGSIb3DQEBCwUA
A4IBAQBJLNoctdzj2t19uXqtJJ5Fyi6hLsJscKkTe7nZlF2/8KlLTva5mplYSOSh
77/qFKVFbl9TxAM9GAEk+7q2I+wHba2r5AGfSAohXxeZt9JgoG1KwIcobT3PkmS0
/7Bj3M9xVYjmknz4JGcprxssQiu6EW4wpjv2jiN3LINx2YJl97WOu3g+d9cqe23i
tfiqGMmlD8ZGch5+XgJV910PHahhWXNECmPX9WUk8DkvyFX114Jr/C7X5DwyU/J+
gXPu8Xns7iDNPg8uxChmxrRWKdWx4jwS+rwBxdkztP6H8iyS9CNaLymoHfohf2/H
LLiL2z6pCD+wzGzkBEbb3mQqAii7
-----END CERTIFICATE-----
`)

	key := []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzj340V7yc0TSOA2MvCcvE1aHcF/jPPj/ptdDGU62ADf6++mp
Cv05ZrmTwhnEgMMBGikj8h9Uoutk286Qcjbd6V6NECPAfx1GbR4C2NZYWrdFxx5J
3L6Qb+bLcXxZ4SZ1HsLCJZQ3ekfyORJbUa491E0+vIyHL0VceplZxe7pWxC74Z3n
UqUqXBnYkjMnrweeeyOPRa/pESs159jn5q6KQq7kV1mqtTKOfUitcIQT36t2QHP3
zALMImZN8/pW4dU1OXXU6O4FfaEorNTIijOFwq3gfuKiBrQ1Lpy0OIHvAyQhI2wf
6VowkHMTbftf4+hn8LkPX9tupPVi8xE3+qeVbQIDAQABAoIBAB1Ovyn5hRiOQhVH
D6W5J75mwG5eoesLM0EhO96/yas0SU09AhGWtG59lpqxkLP/gguWpw4EF8HjE30M
2IfydgxwrDkL65HkthpRdnQb2YhinN7T1gkxZ1MPh/+DfT46edA6Ot6eNlgBG1Gy
4xkzWoPtyil/CsLT53Vgj1pKPgz7jXjdDn8SOxIRn72sdnzsnFT3dYDiq61sHYmp
S8RRT86bTiazvgXVpPeDrF1sx9x8HGjhqynX1Kkxq1w2hRCgOwkCTupNyM6m/Rj0
RiD5BDIyXC5VTKZfnnI4JP3h//MhE5+rojVHCFEeCAnqICq6fsW/h0gpXxmoSmC+
94OvbV0CgYEA50SkkRAkVkArqaPu9G7E5OAqAN2lmTtDmXgCiisLgLuzKbFoBsBw
2JKmrk0/UWjFp51OFJhS5y8vi6I/RKzRaWODTFOGy58ykbAeHg3Lz1sAg6T8zbqX
WKv6qM5HiYugxpQSBlraaRfe9COaxqCS8Ix4i7TtmNxonoCl+ZWiepMCgYEA5Ewy
xya/VVoB+vEennkMkNL5fQY0PmfDoTnax68goOiQcsQE6txGC6YPGgTOsFed2GVa
DzJlMcCLKS8uWZ47hk+TCmwYqTZtm2oUdEynV/ntkLr1xecAt9habm1408ZETVFG
pC7xUVJOGtbBoXVmTZMxq3O22dnUUaBs1LYHr/8CgYEAurWyXuM3UuLv3T9adcDP
+S/4+UX4oeM0yjwXYNErsjzXgnuVzo2jDVYod2QqEGGT4aSgGwR3Oengas0MYzda
wcjzgbWVh+L5AqG7Tuw4dSm1GpMi/jz8XzxJW+td2e/+VxPIEZVb66i3+UadeGq+
9rGRyMjDYbvgQsb+OKfTyz0CgYEAtZAoPhyBxIad1o5W1J/er3sqchUyDYOGoT2a
0n5kC7SJ1MwyQPrINlFt5zp1iudToJsSYc2pj0TbJ/je+uUN4AZ/IaXQgxrHVvep
psijuiMJnbYi6q6J8qx9Vx93Ha2r+nWuIbs3rn4voca0Hg15PC7ZLMsNW/qSkgxt
IUpXmM8CgYEAlc/MOC1WiBpzMgzY5HZ8y/HBe/YoiGYmneqRSN1iku8uPVXSs/An
7legzs9JrQccFqI+pgelDeHOkYMUj4YSQb0n9lzY9ZDDK2OM3xRgzcHvbl1sHcf2
faeZCyUByN54idvbZrDOUemZSnWTbaYibHmyTZgZYPZFMO5IEIfb4pc=
-----END RSA PRIVATE KEY-----`)

	cer, err := tls.X509KeyPair(cert, key)

	if err != nil {
		log.Fatal(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}

	var broadcast = make(chan string, 100)

	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/scan", ScanServer)
	http.HandleFunc("/ws", wsHandler)
	go ReadCards(broadcast)
	go echo(broadcast)

	server := &http.Server{Addr: "127.0.0.1:8081", TLSConfig: config}
	log.Fatal(server.ListenAndServeTLS("", ""))
}
