wget -O vpn.zip https://support.fastestvpn.com/download/fastestvpn_ovpn/
unzip vpn.zip
rm -rf tcp_files
mv udp_files vpn
python3.11 setup-vpn.py