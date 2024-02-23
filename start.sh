cd /etc/openvpn && openvpn $(ls /etc/openvpn/*.ovpn | shuf -n 1) &
sleep 10
addgroup -g $GID lightningbot && adduser -D -u $UID -G lightningbot lightningbot
su lightningbot && /bin/LB-Selfbot