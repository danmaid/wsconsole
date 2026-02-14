# wsconsole Debian Package Build

This directory contains the skeleton for building a .deb package.

## Build Instructions

1. Build the Go binary:
   ```bash
   cd ../..
   GOOS=linux GOARCH=amd64 go build -o wsconsole ./cmd/wsconsole
   ```

2. Prepare the package structure:
   ```bash
   mkdir -p packaging/deb/usr/local/bin
   mkdir -p packaging/deb/usr/local/share/wsconsole/static
   mkdir -p packaging/deb/etc/systemd/system
   mkdir -p packaging/deb/etc/polkit-1/rules.d
   
   cp wsconsole packaging/deb/usr/local/bin/
   cp deploy/static/index.html packaging/deb/usr/local/share/wsconsole/static/
   cp deploy/systemd/wsconsole.service packaging/deb/etc/systemd/system/
   cp deploy/polkit/10-wsconsole.rules packaging/deb/etc/polkit-1/rules.d/
   ```

3. Set permissions:
   ```bash
   chmod 755 packaging/deb/DEBIAN/postinst
   chmod 755 packaging/deb/DEBIAN/prerm
   chmod 755 packaging/deb/usr/local/bin/wsconsole
   ```

4. Build the package:
   ```bash
   dpkg-deb --build packaging/deb wsconsole_1.0.0_amd64.deb
   ```

### Makefile を使う場合

```bash
make deb VERSION=1.0.0 DEB_ARCH=amd64
```

## Installation

```bash
sudo dpkg -i wsconsole_1.0.0_amd64.deb
sudo systemctl start wsconsole
```
