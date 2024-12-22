#### Piko Tunnel

Wireguard based Tunnel to dynamically allocate IP addresses to clients.
It also allows to write communication rules between peers.

#### API Documentation

Open `apidocs` folder in Bruno to see the API documentation.

#### Environment variables

- `SERVER_ADDRESS`: The address to run the server on. If not set, the server will run on `:8080`.
- `WG_MTU`: The MTU to set on the wg0 interface. If not set, the MTU will be set to 1420.

#### Installation

1. Build the binary

```bash
go build -o pikotunnel .
```

2. Move the binary to `/usr/bin`

3. Generate wireguard private key and public key

```bash
# Generate private key
wg genkey > wg_private_key

# Generate public key
wg pubkey < wg_private_key > wg_public_key
```

4. Create folder in `/home/pikotunnel`

5. Write the configuration file `config.json`

```json
{
  "api_token": "your_api_token",
  "wireguard_subnet": "10.0.0.1/16",
  "wireguard_relay_server_public_ip": "x.x.x.x",
  "wireguard_listen_port": 51820,
  "wireguard_private_key": "CO8FfvcsA30LzMB+1q5qe5u9URUKQ7dAviTIfKQnAWU=",
  "wireguard_public_key": "lDxKubEHueyRyM9POkruqhjcL6ADRSmUDsSnvq4/8Ts="
}
```

6. Write systemd service file `pikotunnel.service`

```
[Unit]
Description=Piko Tunnel Service
After=multi-user.target

[Service]
WorkingDirectory=/home/pikotunnel
ExecStart=/usr/bin/pikotunnel server
Type=simple
Restart=always

[Install]
WantedBy=multi-user.target
```

7. Enable and start the service

```bash
sudo systemctl enable pikotunnel.service
sudo systemctl start pikotunnel.service
```
