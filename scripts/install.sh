#!/bin/bash

#-------------------------
# move go-alpr program to "/usr/local/bin/go-alpr/" by hand
#-------------------------

sudo apt-get update && sudo apt-get install -y openalpr openalpr-daemon openalpr-utils libopenalpr-dev

#-------------------------
# Create /usr/local/bin/go-alpr/config.yaml
#-------------------------
echo "
alpr:
    location: /usr/bin/alpr
    stream: http://192.168.24.82/mjpg/video.mjpg
    confidence: 85
    lost: 10000
    scanTime: 2000
mqtt:
    host: my.mqtt.host
    port: 1883
    clientId: lprToGost8345
    streamId: 57" > /usr/local/bin/go-alpr/config.yaml

#-------------------------
# Create /etc/systemd/system/go-alpr.service to run go-alpr as a service
#-------------------------
echo "
[Unit]
Description=go-alpr
After=syslog.target network.target

[Service]
ExecStart=/usr/local/bin/go-alpr/go-alpr -config /usr/local/bin/go-alpr/config.yaml
Restart=on-failure
KillSignal=SIGINT
SyslogIdentifier=go-alpr
StandardOutput=syslog
# non-root user to run as, change user and group to your liking
WorkingDirectory=/home/geodan/
User=geodan
Group=geodan
[Install]
WantedBy=multi-user.target" > /etc/systemd/system/go-alpr.service

#-------------------------
# Enable go-alpr service start on boot
#-------------------------
sudo systemctl daemon-reload
sudo systemctl enable go-alpr 
sudo systemctl start go-alpr