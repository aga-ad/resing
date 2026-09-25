package main

import "fmt"

func tcpTemplate(params map[string]string) (string, error) {
	template :=
		`
{
    "log": {
        "disabled": true
    },
    "inbounds": [
        {
            "type": "socks",
            "tag": "socks-in",
            "listen": "127.0.0.1",
            "listen_port": 1080,
            "users": []
        }
    ],
    "outbounds": [
        {
            "type": "vless",
            "tag": "vless-out",
            "server": "%s",
            "server_port": %s,
            "uuid": "%s",
            "flow": "xtls-rprx-vision",
            "tls": {
                    "enabled": true,
                    "server_name": "%s",
                    "utls": {
                    "enabled": true,
                    "fingerprint": "%s"
                },
                "reality": {
                    "enabled": true,
                    "public_key": "%s"
                }
            },
            "packet_encoding": "xudp"
        }
    ],
    "route": {
        "rules": [
            {
                "inbound": "socks-in",
                "outbound": "vless-out"
            }
        ]
    }
}	
`
	for _, key := range []string{"host", "port", "user", "sni", "fp", "pbk"} {
		if params[key] == "" {
			return "", fmt.Errorf("missing %s", key)
		}
	}

	return fmt.Sprintf(template, params["host"], params["port"], params["user"], params["sni"], params["fp"], params["pbk"]), nil
}

func xhttpTemplate(params map[string]string) (string, error) {
	template :=
		`
{
    "log": {
        "disabled": true
    },
    "inbounds": [
        {
            "type": "socks",
            "tag": "socks-in",
            "listen": "127.0.0.1",
            "listen_port": 1080,
            "users": []
        }
    ],
    "outbounds": [
        {
            "type": "vless",
            "tag": "vless-out",
            "server": "%s",
            "server_port": %s,
            "uuid": "%s",
            "tls": {
                "enabled": true,
                "server_name": "%s",
                "alpn": [
                    "h2",
                    "http/1.1"
                ],
                "reality": {
                    "enabled": true,
                    "public_key": "%s"
                },
                "utls": {
                    "enabled": true,
                    "fingerprint": "%s"
                }
            },
            "transport": {
                "type": "xhttp",
                "path": "%s",
                "mode": "auto",
                "x_padding_bytes": "100-1000",
                "host": "%s"
            }
        }
    ],
    "route": {
        "rules": [
            {
                "inbound": "socks-in",
                "outbound": "vless-out"
            }
        ]
    }
}
`

	for _, key := range []string{"host", "port", "user", "sni", "pbk", "fp", "path"} {
		if params[key] == "" {
			return "", fmt.Errorf("missing %s", key)
		}
	}

	return fmt.Sprintf(template, params["host"], params["port"], params["user"], params["sni"], params["pbk"], params["fp"], params["path"], params["sni"]), nil
}
