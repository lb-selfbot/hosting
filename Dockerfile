FROM alpine:latest

COPY LB-Selfbot /bin
COPY start.sh /bin

# Install VPN
RUN apk add --no-cache bash openvpn sudo
COPY ./vpn /etc/openvpn

# Run the bot
CMD ["/bin/bash", "-c", "/bin/start.sh"]
