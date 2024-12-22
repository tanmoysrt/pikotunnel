#### Piko Tunnel

Wireguard based Tunnel to dynamically allocate IP addresses to clients.
It also allows to write communication rules between peers.

It's just a wrapper around wireguard and iptables to provide simple api to manage wireguard peers and allow to allocate communicatio rules betwwen peers.
For example, you can add a real user and a container as a peer and then create a rule, so that the user can access the container easily and later you can revoke the access rule as well.

#### API Documentation

Open `apidocs` folder in Bruno to see the API documentation.

#### Environment variables

- `SERVER_ADDRESS`: The address to run the server on. If not set, the server will run on `:8080`.
- `WG_MTU`: The MTU to set on the wg0 interface. If not set, the MTU will be set to 1420.

#### Installation

1. Install Build Dependencies

```bash
# Install gcc and wget
apt install -y build-essential wget

# Install golang
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

2. Build the binary

```bash
CGO_ENABLED=1 go build -o pikotunnel .
```

3. Move the binary to `/usr/bin`

4. Generate wireguard private key and public key

```bash
# Generate private key
wg genkey > wg_private_key

# Generate public key
wg pubkey < wg_private_key > wg_public_key
```

5. Create folder in `/home/pikotunnel`

6. Write the configuration file `config.json`

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

7. Write systemd service file `/etc/systemd/system/pikotunnel.service`

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

8. Enable and start the service

```bash
systemctl enable pikotunnel.service
systemctl start pikotunnel.service
```

9. Done !
