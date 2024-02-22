cd /etc/openvpn && openvpn $(ls /etc/openvpn/*.ovpn | shuf -n 1) &
sleep 10
su lightningbot && ./LB-Selfbot