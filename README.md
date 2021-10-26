# BitterJohn
Server and relay side infrastructure for RDA.

## Usage

### install

```bash
sudo ./BitterJohn install -g
systemctl enable --now BitterJohn
```

### upgrade

```bash
sudo ./BitterJohn install
sudo systemctl daemon-reload
sudo systemctl restart BitterJohn
```

## Troubleshot

1. User systemd service will be killed after logout. See [stackexchange](https://unix.stackexchange.com/questions/521538/system-service-running-as-user-is-terminated-on-logout).
